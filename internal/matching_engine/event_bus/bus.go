package eventbus

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/config"
	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/google/uuid"
)

// Event is a fact the matching engine has already applied to the book. All events of a
// book go through one bus, so they are settled in the order the engine produced them:
// a cancel is never settled before the trades the order took part in.
//
// The engine publishes exactly one event per command, carrying the command's ID, so
// settlement can apply the command and remove it from the command log.
type Event interface {
	isEvent()
}

// OrderPlaced is published for every placed order, with the trades it produced (none
// when the order went to the book without matching).
type OrderPlaced struct {
	CommandID uuid.UUID
	Trades    []entity.Trade
}

// OrderCancelled is published when the order was removed from the book.
type OrderCancelled struct {
	CommandID uuid.UUID
}

// CancelRejected is published when the order to cancel wasn't in the book: it was
// filled, or already cancelled, before the command reached the engine.
type CancelRejected struct {
	CommandID uuid.UUID
}

func (OrderPlaced) isEvent()    {}
func (OrderCancelled) isEvent() {}
func (CancelRejected) isEvent() {}

type EventBus struct {
	queue chan Event
}

func (b *EventBus) PublishOrderPlaced(ctx context.Context, commandID uuid.UUID, trades []entity.Trade) error {
	return b.publish(ctx, OrderPlaced{CommandID: commandID, Trades: trades})
}

func (b *EventBus) PublishOrderCancelled(ctx context.Context, commandID uuid.UUID) error {
	return b.publish(ctx, OrderCancelled{CommandID: commandID})
}

func (b *EventBus) PublishCancelRejected(ctx context.Context, commandID uuid.UUID) error {
	return b.publish(ctx, CancelRejected{CommandID: commandID})
}

func (b *EventBus) Consume(ctx context.Context) (Event, error) {
	select {
	case event := <-b.queue:
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *EventBus) publish(ctx context.Context, event Event) error {
	select {
	case b.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewEventBus(cfg *config.EventBusConfig) *EventBus {
	return &EventBus{
		queue: make(chan Event, cfg.BufferSize),
	}
}
