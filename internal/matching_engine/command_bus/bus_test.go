package commandbus

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

// commandOrderID returns the ID of the order the command is about.
func commandOrderID(t *testing.T, command Command) uuid.UUID {
	t.Helper()

	switch c := command.(type) {
	case PlaceOrder:
		return c.Order.ID
	case CancelOrder:
		return c.OrderID
	}
	t.Fatalf("unexpected command type %T", command)
	return uuid.Nil
}

// Two sender goroutines send commands, but their turns are coordinated via a
// baton channel so the resulting interleaving is deterministic: s1, s2, s1, s2.
// The baton is unbuffered, so each handoff only completes once the other side
// is actually waiting for it, which serializes the four sends into one
// strict, known order even though they come from two different goroutines.
// Places and cancels are mixed: they share the bus precisely so that the engine
// receives them in the order they were sent.
func TestCommandBus_MultipleSendersPreserveHandoffOrder(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 10, SendTimeout: testTimeout})

	order1, order2 := newTestOrder(), newTestOrder()

	baton := make(chan struct{})
	done := make(chan error, 2)

	go func() {
		if err := bus.SendPlaceOrder(context.Background(), uuid.New(), order1); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let s2 place order2
		<-baton             // wait for order2 to be placed
		if err := bus.SendCancelOrder(context.Background(), uuid.New(), order1.ID); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let s2 cancel order2
		done <- nil
	}()

	go func() {
		<-baton // wait for order1 to be placed
		if err := bus.SendPlaceOrder(context.Background(), uuid.New(), order2); err != nil {
			done <- err
			return
		}
		baton <- struct{}{} // let s1 cancel order1
		<-baton             // wait for order1 to be cancelled
		done <- bus.SendCancelOrder(context.Background(), uuid.New(), order2.ID)
	}()

	require.NoError(t, <-done)
	require.NoError(t, <-done)

	want := []Command{
		PlaceOrder{},
		PlaceOrder{},
		CancelOrder{},
		CancelOrder{},
	}
	wantIDs := []uuid.UUID{order1.ID, order2.ID, order1.ID, order2.ID}
	for i := range want {
		command, err := bus.Receive(context.Background())
		require.NoError(t, err)
		assert.IsType(t, want[i], command)
		assert.Equal(t, wantIDs[i], commandOrderID(t, command))
	}
}

// Once the queue's buffer is full and nothing drains it, a send must not
// block forever: it should give up after sendTimeout and report that timeout
// as the error, without ever writing the second command into the queue.
func TestCommandBus_Send_RespectsSendTimeout(t *testing.T) {
	const sendTimeout = 30 * time.Millisecond
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: sendTimeout})

	first := newTestOrder()
	require.NoError(t, bus.SendPlaceOrder(context.Background(), uuid.New(), first))

	start := time.Now()
	err := bus.SendCancelOrder(context.Background(), uuid.New(), first.ID)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.GreaterOrEqual(t, elapsed, sendTimeout)

	// The only command that made it into the queue is the first one.
	receiveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	got, err := bus.Receive(receiveCtx)
	require.NoError(t, err)
	require.IsType(t, PlaceOrder{}, got)
	assert.Equal(t, first.ID, commandOrderID(t, got))

	receiveCtx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	_, err = bus.Receive(receiveCtx2)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// A send blocks on a full queue; cancelling the caller's context from
// outside must unblock it immediately with context.Canceled, independent of
// the (much longer) send timeout.
func TestCommandBus_Send_ReturnsErrorWhenContextCancelledExternally(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 0, SendTimeout: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- bus.SendPlaceOrder(ctx, uuid.New(), newTestOrder())
	}()

	time.Sleep(20 * time.Millisecond) // let the send reach the select
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(testTimeout):
		t.Fatal("SendPlaceOrder did not return after context was cancelled")
	}
}

// Receive blocks on an empty queue; cancelling the caller's context
// from outside must unblock it immediately with context.Canceled.
func TestCommandBus_Receive_ReturnsErrorWhenContextCancelledExternally(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 0, SendTimeout: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := bus.Receive(ctx)
		errCh <- err
	}()

	time.Sleep(20 * time.Millisecond) // let Receive reach the select
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(testTimeout):
		t.Fatal("Receive did not return after context was cancelled")
	}
}
