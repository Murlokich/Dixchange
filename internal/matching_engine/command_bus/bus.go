package commandbus

import (
	"context"
	"time"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	"github.com/google/uuid"
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

type CommandBus struct {
	sendTimeout time.Duration
	queue       chan Command
}

func (b *CommandBus) SendPlaceOrder(ctx context.Context, commandID uuid.UUID, entityOrder entity.Order) error {
	return b.send(ctx, PlaceOrder{CommandID: commandID, Order: orderbook.NewOrderFromEntity(entityOrder)})
}

func (b *CommandBus) SendCancelOrder(ctx context.Context, commandID, orderID uuid.UUID) error {
	return b.send(ctx, CancelOrder{CommandID: commandID, OrderID: orderID})
}

func (b *CommandBus) Receive(ctx context.Context) (Command, error) {
	select {
	case command := <-b.queue:
		return command, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *CommandBus) send(ctx context.Context, command Command) error {
	ctx, cancel := context.WithTimeout(ctx, b.sendTimeout)
	defer cancel()

	select {
	case b.queue <- command:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewCommandBus(cfg *config.CommandBusConfig) *CommandBus {
	return &CommandBus{
		queue:       make(chan Command, cfg.BufferSize),
		sendTimeout: cfg.SendTimeout,
	}
}
