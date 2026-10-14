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

// Once every slot is taken, GetWriteAccess must not block forever: it gives up after
// sendTimeout and reports overload. Receiving the command frees its slot.
func TestCommandBus_GetWriteAccess_RespectsSendTimeout(t *testing.T) {
	const sendTimeout = 30 * time.Millisecond
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: sendTimeout})

	first := newTestOrder()
	ticket, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err)
	require.NoError(t, ticket.SendPlaceOrder(uuid.New(), first))

	start := time.Now()
	_, err = bus.GetWriteAccess(context.Background())
	elapsed := time.Since(start)

	require.ErrorIs(t, err, entity.ErrCommandBusOverloaded)
	assert.GreaterOrEqual(t, elapsed, sendTimeout)

	got, err := bus.Receive(context.Background())
	require.NoError(t, err)
	assert.Equal(t, first.ID, commandOrderID(t, got))

	_, err = bus.GetWriteAccess(context.Background())
	require.NoError(t, err, "receiving the command must free its slot")
}

// GetWriteAccess blocks while every slot is taken; cancelling the caller's context from
// outside must unblock it immediately with context.Canceled, independent of the (much
// longer) send timeout.
func TestCommandBus_GetWriteAccess_ReturnsErrorWhenContextCancelledExternally(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: time.Minute})
	_, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, accessErr := bus.GetWriteAccess(ctx)
		errCh <- accessErr
	}()

	time.Sleep(20 * time.Millisecond) // let GetWriteAccess reach the select
	cancel()

	select {
	case err = <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(testTimeout):
		t.Fatal("GetWriteAccess did not return after context was cancelled")
	}
}

// Releasing an unsent ticket gives its slot back; releasing it again does nothing, so it
// can't free a slot held by someone else.
func TestAccessTicket_ReleaseFreesSlotOnce(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: 30 * time.Millisecond})

	first, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err)
	require.NoError(t, first.ReleaseWriteAccess())

	second, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err, "the released slot must be free again")

	require.NoError(t, first.ReleaseWriteAccess())
	_, err = bus.GetWriteAccess(context.Background())
	require.ErrorIs(t, err, entity.ErrCommandBusOverloaded, "a second release must not free the second ticket's slot")

	require.NoError(t, second.ReleaseWriteAccess())
}

// A ticket sends at most one command, and once it has sent, its slot belongs to the
// command: a release does nothing and only Receive frees the slot.
func TestAccessTicket_SendsOnce(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: 30 * time.Millisecond})

	ticket, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err)
	require.NoError(t, ticket.SendPlaceOrder(uuid.New(), newTestOrder()))

	require.ErrorIs(t, ticket.SendCancelOrder(uuid.New(), uuid.New()), ErrTicketUsed)
	require.NoError(t, ticket.ReleaseWriteAccess())

	_, err = bus.GetWriteAccess(context.Background())
	require.ErrorIs(t, err, entity.ErrCommandBusOverloaded, "the slot must stay taken until Receive")
}

// A ticket whose send finds the queue full reports the broken invariant and keeps its
// slot, so the release still gives it back.
func TestAccessTicket_SendReportsFullQueue(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: testTimeout})

	ticket, err := bus.GetWriteAccess(context.Background())
	require.NoError(t, err)
	bus.commandQueue <- PlaceOrder{} // a command that bypassed the slots

	require.ErrorIs(t, ticket.SendPlaceOrder(uuid.New(), newTestOrder()), ErrAccessQueueAsync)
	require.NoError(t, ticket.ReleaseWriteAccess())
}

// A command without a slot means the slots and the queue diverged: Receive reports it
// instead of blocking, so the engine stops.
func TestCommandBus_Receive_ReportsCommandWithoutSlot(t *testing.T) {
	bus := NewCommandBus(&config.CommandBusConfig{BufferSize: 1, SendTimeout: testTimeout})
	bus.commandQueue <- PlaceOrder{}

	_, err := bus.Receive(context.Background())
	require.ErrorIs(t, err, ErrAccessQueueAsync)
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
