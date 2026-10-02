package engine

import (
	"context"
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	commandbus "github.com/dixchange/Monolith/internal/matching_engine/command_bus"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	pkgerrors "github.com/pkg/errors"
)

type OrderBook interface {
	AddOrder(order *orderbook.Order) ([]orderbook.Trade, error)
	CancelOrder(id uuid.UUID) (bool, error)
	Pair() entity.Pair
}

type CommandBus interface {
	Receive(ctx context.Context) (commandbus.Command, error)
}

type EventBus interface {
	PublishOrderPlaced(ctx context.Context, commandID uuid.UUID, trades []entity.Trade) error
	PublishOrderCancelled(ctx context.Context, commandID uuid.UUID) error
	PublishCancelRejected(ctx context.Context, commandID uuid.UUID) error
}

type MatchingEngine struct {
	book       OrderBook
	commandBus CommandBus
	eventBus   EventBus
	logger     pkg.Logger
}

func (m *MatchingEngine) Run(ctx context.Context) {
	for {
		command, err := m.commandBus.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			m.logger.Errorf("commandBus Receive: %v", err)
			continue
		}

		var orderID uuid.UUID
		switch c := command.(type) {
		case commandbus.PlaceOrder:
			orderID = c.Order.ID
			err = m.placeOrder(ctx, c.CommandID, c.Order)
		case commandbus.CancelOrder:
			orderID = c.OrderID
			err = m.cancelOrder(ctx, c.CommandID, c.OrderID)
		default:
			m.logger.Errorf("unknown command type %T", command)
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			// TODO(recovery): the command stays in the command log, but the book may
			// already have applied it (publish failed) or be corrupted (book error), so
			// the book and the DB diverge. Reload the engine from the orders and the
			// command log instead of carrying on.
			m.logger.WithField("order_id", orderID).Errorf("handle %T: %v", command, err)
		}
	}
}

// placeOrder matches the order against the book and publishes the trades it produced.
// The event is published even without trades, so the order reaches the orders.
func (m *MatchingEngine) placeOrder(ctx context.Context, commandID uuid.UUID, order orderbook.Order) error {
	trades, err := m.book.AddOrder(&order)
	if err != nil {
		return pkgerrors.Wrap(err, "book add order")
	}
	entityTrades := orderbookToEntityTrades(trades, m.book.Pair())
	if err = m.eventBus.PublishOrderPlaced(ctx, commandID, entityTrades); err != nil {
		return pkgerrors.Wrap(err, "eventBus PublishOrderPlaced")
	}
	return nil
}

// cancelOrder removes the order from the book and publishes the cancellation, so the
// funds still reserved for it get released.
func (m *MatchingEngine) cancelOrder(ctx context.Context, commandID, orderID uuid.UUID) error {
	cancelled, err := m.book.CancelOrder(orderID)
	if err != nil {
		// TODO(recovery): an error here means the book's internal state is broken (a list node that
		// isn't an order, a price level that can't be deleted), not a bad request. Decide
		// whether the engine should halt instead of carrying on with a corrupted book.
		return pkgerrors.Wrap(err, "book cancel order")
	}
	if !cancelled {
		// Normal outcome: the order was filled, or already cancelled, before this command
		// reached the engine. There is nothing to release, but the command is published
		// anyway so settlement removes it from the command log.
		m.logger.WithField("order_id", orderID).Infof("order to cancel is not in the book")
		if err = m.eventBus.PublishCancelRejected(ctx, commandID); err != nil {
			return pkgerrors.Wrap(err, "eventBus PublishCancelRejected")
		}
		return nil
	}

	if err = m.eventBus.PublishOrderCancelled(ctx, commandID); err != nil {
		return pkgerrors.Wrap(err, "eventBus PublishOrderCancelled")
	}
	return nil
}

func NewMatchingEngine(
	book OrderBook,
	commandBus CommandBus,
	eventBus EventBus,
	logger pkg.Logger,
) *MatchingEngine {
	return &MatchingEngine{
		book:       book,
		commandBus: commandBus,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// The book matches a single pair, so the pair is stamped onto every trade it produced
// instead of being carried by each order and trade inside the book.
func orderbookToEntityTrades(trades []orderbook.Trade, pair entity.Pair) []entity.Trade {
	entityTrades := make([]entity.Trade, len(trades))
	for i := range trades {
		entityTrades[i] = entity.Trade{
			MarketPair: pair,
			Ask:        trades[i].Ask,
			Bid:        trades[i].Bid,
			Price:      trades[i].Price,
			Quantity:   trades[i].Quantity,
		}
	}
	return entityTrades
}
