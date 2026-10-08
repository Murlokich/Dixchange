package engine

import (
	"context"

	"github.com/pkg/errors"

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

type OrderRepository interface {
	GetAllOpenOrders(ctx context.Context) ([]entity.Order, error)
	GetAllCommands(ctx context.Context) ([]entity.OrderCommand, error)
}

type MatchingEngine struct {
	book       OrderBook
	commandBus CommandBus
	eventBus   EventBus
	orderRepo  OrderRepository
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

		orderID, err := m.processCommand(ctx, command)
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

func NewMatchingEngine(
	book OrderBook,
	commandBus CommandBus,
	eventBus EventBus,
	orderRepo OrderRepository,
	logger pkg.Logger,
) *MatchingEngine {
	return &MatchingEngine{
		book:       book,
		commandBus: commandBus,
		eventBus:   eventBus,
		orderRepo:  orderRepo,
		logger:     logger,
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

func (m *MatchingEngine) LoadOrderBook(ctx context.Context) error {
	m.logger.Infof("order book loading has been started")
	if err := m.loadOpenOrders(ctx); err != nil {
		return errors.Wrap(err, "loaodOpenOrders")
	}
	m.logger.Infof("open orders have been loaded successfully")
	if err := m.applyRequestLog(ctx); err != nil {
		return errors.Wrap(err, "applyRequestLog")
	}
	m.logger.Infof("request log has been successfully applied")
	return nil
}

func (m *MatchingEngine) loadOpenOrders(ctx context.Context) error {
	orders, err := m.orderRepo.GetAllOpenOrders(ctx)
	if err != nil {
		return errors.Wrap(err, "order repo GetAllOpenOrders")
	}
	for i := range orders {
		order := orderbook.NewOrderFromEntity(orders[i])
		trades, err := m.book.AddOrder(&order)
		if err != nil {
			return errors.Wrap(err, "book AddOrder")
		}
		if len(trades) != 0 {
			return errors.Errorf("unexpected trade produced by resting orders: %v", trades)
		}
	}
	return nil
}

func (m *MatchingEngine) applyRequestLog(ctx context.Context) error {
	commands, err := m.orderRepo.GetAllCommands(ctx)
	if err != nil {
		return errors.Wrap(err, "order repo GetAllCommands")
	}
	for i := range commands {
		cmd, err := logEntryToBusCommand(commands[i])
		if err != nil {
			return errors.Wrap(err, "logEntryToBusCommand")
		}
		orderID, err := m.processCommand(ctx, cmd)
		if err != nil {
			m.logger.WithField("order_id", orderID).Errorf("processCommand %v: %v", cmd, err)
			return errors.Wrap(err, "processCommand")
		}
	}
	return nil
}

// processCommand returns orderID for logging purposes
func (m *MatchingEngine) processCommand(ctx context.Context, command commandbus.Command) (uuid.UUID, error) {
	var orderID uuid.UUID
	var err error
	switch c := command.(type) {
	case commandbus.PlaceOrder:
		orderID = c.Order.ID
		err = m.placeOrder(ctx, c.CommandID, c.Order)
	case commandbus.CancelOrder:
		orderID = c.OrderID
		err = m.cancelOrder(ctx, c.CommandID, c.OrderID)
	default:
		err = errors.Errorf("unknown command type %T", command)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return orderID, nil
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

func logEntryToBusCommand(cmd entity.OrderCommand) (commandbus.Command, error) {
	switch cmd.Type {
	case entity.OrderCommandCreate:
		order, err := cmd.CreatedOrder()
		if err != nil {
			return nil, errors.Wrap(err, "command CreatedOrder")
		}
		return commandbus.PlaceOrder{
			CommandID: cmd.ID,
			Order:     orderbook.NewOrderFromEntity(order),
		}, nil

	case entity.OrderCommandCancel:
		return commandbus.CancelOrder{
			CommandID: cmd.ID,
			OrderID:   cmd.OrderID,
		}, nil
	default:
		return nil, entity.InvalidCommandType{CommandType: string(cmd.Type)}
	}
}
