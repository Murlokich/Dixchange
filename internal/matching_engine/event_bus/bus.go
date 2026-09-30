package eventbus

import (
	"context"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
)

// Event is a fact the matching engine has already applied to the book. All events of a
// book go through one bus, so they are settled in the order the engine produced them:
// a cancel is never settled before the trades the order took part in.
type Event interface {
	isEvent()
}

type TradesExecuted struct {
	Trades []entity.Trade
}

type OrderCancelled struct {
	OrderID uuid.UUID
}

func (TradesExecuted) isEvent() {}
func (OrderCancelled) isEvent() {}

type EventBus struct {
	queue chan Event
}

func (b *EventBus) PublishTrades(ctx context.Context, trades []entity.Trade) error {
	if len(trades) == 0 {
		return nil
	}
	return b.publish(ctx, TradesExecuted{Trades: trades})
}

func (b *EventBus) PublishOrderCancelled(ctx context.Context, orderID uuid.UUID) error {
	return b.publish(ctx, OrderCancelled{OrderID: orderID})
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
