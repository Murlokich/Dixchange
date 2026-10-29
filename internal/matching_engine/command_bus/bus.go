package commandbus

import (
	"context"
	"errors"
	"time"

	"github.com/Murlokich/Dixchange/internal/config"
	"github.com/Murlokich/Dixchange/internal/domain/entity"
	orderbook "github.com/Murlokich/Dixchange/internal/matching_engine/order_book"
	"github.com/Murlokich/Dixchange/pkg"
	"github.com/google/uuid"
)

var (
	// ErrAccessQueueAsync means the slots no longer match the commands in the queue. It is
	// a bug, never overload: a slot always guarantees room in the queue.
	ErrAccessQueueAsync = errors.New("access and command queue are not synchronized")
	// ErrTicketUsed is returned when a ticket that already sent or released tries to send.
	ErrTicketUsed = errors.New("access ticket already used")
)

// Command is a request to the matching engine, which may still reject it. Every command
// for a book goes through one bus, so the engine sees them in a single order: a cancel
// can't overtake the placement of the order it cancels. CommandID is the ID of the
// command in the command log; the engine passes it on in the event it publishes.
type Command interface {
	isCommand()
}

type PlaceOrder struct {
	CommandID uuid.UUID
	Order     orderbook.Order
}

type CancelOrder struct {
	CommandID uuid.UUID
	OrderID   uuid.UUID
}

func (PlaceOrder) isCommand()  {}
func (CancelOrder) isCommand() {}

// AccessTicket holds one slot of the command queue. Only GetWriteAccess creates it.
type AccessTicket struct {
	// used is set once the ticket sent a command or released its slot: either way the
	// slot is no longer the ticket's.
	used             bool
	commandQueue     chan<- Command
	writeAccessQueue <-chan struct{}
}

func newAccessTicket(commandQueue chan<- Command, writeAccessQueue <-chan struct{}) *AccessTicket {
	return &AccessTicket{commandQueue: commandQueue, writeAccessQueue: writeAccessQueue}
}

// SendPlaceOrder sends a place command in the ticket's slot.
func (t *AccessTicket) SendPlaceOrder(commandID uuid.UUID, entityOrder entity.Order) error {
	return t.send(PlaceOrder{CommandID: commandID, Order: orderbook.NewOrderFromEntity(entityOrder)})
}

// SendCancelOrder sends a cancel command in the ticket's slot.
func (t *AccessTicket) SendCancelOrder(commandID, orderID uuid.UUID) error {
	return t.send(CancelOrder{CommandID: commandID, OrderID: orderID})
}

// send never waits and ignores cancellation: it runs after the command is committed, so it
// must not be dropped, and the ticket's slot guarantees room in the queue.
func (t *AccessTicket) send(command Command) error {
	if t.used {
		return ErrTicketUsed
	}
	select {
	case t.commandQueue <- command:
		t.used = true
		return nil
	default:
		return ErrAccessQueueAsync
	}
}

// ReleaseWriteAccess gives the slot back if the ticket hasn't sent a command, and does
// nothing otherwise: a sent command's slot is freed by Receive.
func (t *AccessTicket) ReleaseWriteAccess() error {
	if t.used {
		return nil
	}
	t.used = true
	return releaseSlot(t.writeAccessQueue)
}

// releaseSlot frees one slot without waiting: a missing slot means the slots and the
// queue have diverged.
func releaseSlot(writeAccessQueue <-chan struct{}) error {
	select {
	case <-writeAccessQueue:
		return nil
	default:
		return ErrAccessQueueAsync
	}
}

// CommandBus is a bounded queue of commands for the engine. writeAccessQueue counts the
// slots taken: commands in the queue plus tickets that may still send one. It has the
// same capacity as commandQueue, so a ticket's send always finds room.
type CommandBus struct {
	sendTimeout      time.Duration
	writeAccessQueue chan struct{}
	commandQueue     chan Command
}

// Receive takes the next command and frees the slot it held.
func (b *CommandBus) Receive(ctx context.Context) (Command, error) {
	select {
	case command := <-b.commandQueue:
		if err := b.releaseWriteAccess(); err != nil {
			return nil, err
		}
		return command, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *CommandBus) releaseWriteAccess() error {
	return releaseSlot(b.writeAccessQueue)
}

// GetWriteAccess reserves a slot in the queue. Take it before writing anything, so an
// overloaded engine rejects a request before it has side effects. If no slot frees up
// within sendTimeout it returns entity.ErrCommandBusOverloaded; if ctx ends first, ctx's
// error. The caller must defer ReleaseWriteAccess on the returned ticket.
func (b *CommandBus) GetWriteAccess(ctx context.Context) (pkg.AccessTicket, error) {
	// The cause tells our timeout apart from the caller's: both end with DeadlineExceeded.
	ctx, cancel := context.WithTimeoutCause(ctx, b.sendTimeout, entity.ErrCommandBusOverloaded)
	defer cancel()

	select {
	case b.writeAccessQueue <- struct{}{}:
		return newAccessTicket(b.commandQueue, b.writeAccessQueue), nil
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
}

func NewCommandBus(cfg *config.CommandBusConfig) *CommandBus {
	return &CommandBus{
		commandQueue:     make(chan Command, cfg.BufferSize),
		writeAccessQueue: make(chan struct{}, cfg.BufferSize),
		sendTimeout:      cfg.SendTimeout,
	}
}
