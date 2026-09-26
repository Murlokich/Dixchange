package bookwriter

import (
	"context"
	"testing"
	"time"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = time.Second

func newTestOrder() entity.Order {
	return entity.Order{
		ID:              uuid.New(),
		Side:            entity.BuySide,
		PricePerUnit:    100,
		QuantityOrdered: 1,
	}
}

// Two writer goroutines insert orders, but their turns are coordinated via a
// baton channel so the resulting interleaving is deterministic: w1, w2, w1, w2.
// The baton is unbuffered, so each handoff only completes once the other side
// is actually waiting for it, which serializes the four inserts into one
// strict, known order even though they come from two different goroutines.
// This confirms ordering is respected across goroutines too, not just within one.
func TestBookWriter_InsertOrder_MultipleWritersPreserveHandoffOrder(t *testing.T) {
	writer := NewBookWriter(&config.BookWriterConfig{BufferSize: 10, WriteTimeout: testTimeout})

	order1a, order2a := newTestOrder(), newTestOrder()
	order1b, order2b := newTestOrder(), newTestOrder()

	baton := make(chan struct{})
	done := make(chan error, 2)

	go func() {
		if err := writer.InsertOrder(context.Background(), order1a); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let w2 insert order2a
		<-baton             // wait for order2a to be inserted
		if err := writer.InsertOrder(context.Background(), order1b); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let w2 insert order2b
		done <- nil
	}()

	go func() {
		<-baton // wait for order1a to be inserted
		if err := writer.InsertOrder(context.Background(), order2a); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let w1 insert order1b
		<-baton             // wait for order1b to be inserted
		done <- writer.InsertOrder(context.Background(), order2b)
	}()

	require.NoError(t, <-done)
	require.NoError(t, <-done)

	want := []uuid.UUID{order1a.ID, order2a.ID, order1b.ID, order2b.ID}
	got := make([]uuid.UUID, 0, len(want))
	for range want {
		order, err := writer.RetrieveOrder(context.Background())
		require.NoError(t, err)
		got = append(got, order.ID)
	}
	assert.Equal(t, want, got)
}

// Once the queue's buffer is full and nothing drains it, InsertOrder must not
// block forever: it should give up after writeTimeout and report that timeout
// as the error, without ever writing the second order into the queue.
func TestBookWriter_InsertOrder_RespectsWriteTimeout(t *testing.T) {
	const writeTimeout = 30 * time.Millisecond
	writer := NewBookWriter(&config.BookWriterConfig{BufferSize: 1, WriteTimeout: writeTimeout})

	first := newTestOrder()
	require.NoError(t, writer.InsertOrder(context.Background(), first))

	second := newTestOrder()
	start := time.Now()
	err := writer.InsertOrder(context.Background(), second)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.GreaterOrEqual(t, elapsed, writeTimeout)

	// The only order that made it into the queue is the first one.
	retrieveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	got, err := writer.RetrieveOrder(retrieveCtx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, got.ID)

	retrieveCtx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	_, err = writer.RetrieveOrder(retrieveCtx2)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// InsertOrder blocks on a full queue; cancelling the caller's context from
// outside must unblock it immediately with context.Canceled, independent of
// the (much longer) write timeout.
func TestBookWriter_InsertOrder_ReturnsErrorWhenContextCancelledExternally(t *testing.T) {
	writer := NewBookWriter(&config.BookWriterConfig{BufferSize: 0, WriteTimeout: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- writer.InsertOrder(ctx, newTestOrder())
	}()

	time.Sleep(20 * time.Millisecond) // let InsertOrder reach the select
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(testTimeout):
		t.Fatal("InsertOrder did not return after context was cancelled")
	}
}

// RetrieveOrder blocks on an empty queue; cancelling the caller's context
// from outside must unblock it immediately with context.Canceled.
func TestBookWriter_RetrieveOrder_ReturnsErrorWhenContextCancelledExternally(t *testing.T) {
	writer := NewBookWriter(&config.BookWriterConfig{BufferSize: 0, WriteTimeout: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := writer.RetrieveOrder(ctx)
		errCh <- err
	}()

	time.Sleep(20 * time.Millisecond) // let RetrieveOrder reach the select
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(testTimeout):
		t.Fatal("RetrieveOrder did not return after context was cancelled")
	}
}
