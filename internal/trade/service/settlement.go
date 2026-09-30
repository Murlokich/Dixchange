package service

import (
	"context"
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	eventbus "github.com/dixchange/Monolith/internal/matching_engine/event_bus"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	pkgerrors "github.com/pkg/errors"
)

type EventBus interface {
	Consume(ctx context.Context) (eventbus.Event, error)
}

type TradeRepository interface {
	CreateTrade(ctx context.Context, trade entity.Trade) error
}

type OrderRepository interface {
	FillOrder(ctx context.Context, orderID uuid.UUID, quantity int64) error
	GetOrderForUpdate(ctx context.Context, orderID uuid.UUID) (entity.Order, error)
	CancelOrder(ctx context.Context, orderID uuid.UUID) error
}

type WalletRepository interface {
	TransferFunds(ctx context.Context, fromUserID, toUserID uuid.UUID, symbolID int, amount int64) error
	ReleaseReservedFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

// SettlementService applies what the matching engine did to the book onto the database.
// It consumes the engine's events one at a time, in the order they were produced.
type SettlementService struct {
	eventBus   EventBus
	tradeRepo  TradeRepository
	orderRepo  OrderRepository
	walletRepo WalletRepository
	marketRepo MarketRepository
	txManager  TXManager
	logger     pkg.Logger
}

func (s *SettlementService) Run(ctx context.Context) {
	for {
		event, err := s.eventBus.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			s.logger.Errorf("eventBus Consume: %v", err)
			continue
		}

		logger := s.logger
		switch e := event.(type) {
		case eventbus.TradesExecuted:
			err = s.settleTrades(ctx, e.Trades)
		case eventbus.OrderCancelled:
			logger = logger.WithField("order_id", e.OrderID)
			err = s.executeCancellation(ctx, e.OrderID)
		default:
			s.logger.Errorf("unknown event type %T", event)
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			// TODO: the engine has already applied this event to the book, so the book and
			// the DB now diverge (a trade not settled, or a cancelled order still active with
			// its funds reserved). Decide on one strategy for both: halt so the restart
			// reloads the book from the DB, or persist the event and retry.
			logger.Errorf("settle %T: %v", event, err)
		}
	}
}

// settleTrades settles the trades one by one. A failed trade doesn't stop the rest of the
// batch from settling: every failure is returned, joined.
func (s *SettlementService) settleTrades(ctx context.Context, trades []entity.Trade) error {
	var errs []error
	for i := range trades {
		if err := s.executeTrade(ctx, trades[i]); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			errs = append(errs, pkgerrors.Wrapf(err, "execute trade of bid order %s and ask order %s",
				trades[i].Bid.OrderID, trades[i].Ask.OrderID))
		}
	}
	return errors.Join(errs...)
}

// executeCancellation marks the order cancelled and releases the funds still reserved
// for its unfilled part, within one transaction. Events are settled in order, so every
// trade of the order is already settled and its filled quantity in the DB is final.
func (s *SettlementService) executeCancellation(ctx context.Context, orderID uuid.UUID) error {
	fn := func(ctx context.Context) error {
		order, err := s.orderRepo.GetOrderForUpdate(ctx, orderID)
		if err != nil {
			return pkgerrors.Wrap(err, "order repo get order for update")
		}

		// The engine only publishes a cancellation for an order it removed from the book,
		// so the order must still be active. Skipping instead of failing keeps a repeated
		// event from releasing the funds twice, which matters once events can be
		// redelivered (at least once delivery through a broker).
		if !order.IsActive() {
			s.logger.WithField("order_id", orderID).
				Errorf("cancellation of order in status %s skipped: order isn't active", order.Status)
			return nil
		}

		pair, ok := s.marketRepo.GetPair(order.MarketPair.ID)
		if !ok {
			return pkgerrors.Errorf("market repo pair %d doesn't exist", order.MarketPair.ID)
		}
		order.MarketPair = pair

		symbol, err := order.GetDebitAsset()
		if err != nil {
			return pkgerrors.Wrap(err, "get debit asset")
		}
		amount, err := order.GetDebitRemainingAmount()
		if err != nil {
			return pkgerrors.Wrap(err, "get debit remaining amount")
		}

		if err = s.orderRepo.CancelOrder(ctx, orderID); err != nil {
			return pkgerrors.Wrap(err, "order repo cancel order")
		}
		// Fails when less than amount is reserved, which the current design rules out:
		// seeing it means the reservation and the order have diverged.
		if err = s.walletRepo.ReleaseReservedFunds(ctx, order.UserID, symbol.ID, amount); err != nil {
			return pkgerrors.Wrap(err, "wallet repo release reserved funds")
		}

		return nil
	}

	if err := s.txManager.WithTX(ctx, fn); err != nil {
		return pkgerrors.Wrap(err, "tx manager with tx")
	}

	return nil
}

// executeTrade settles a single trade: it records the trade, moves the funds between
// the participants and updates the filled quantity of both orders. The whole
// settlement is applied within one transaction, so a trade is either fully settled
// or not settled at all.
func (s *SettlementService) executeTrade(ctx context.Context, trade entity.Trade) error {
	// trade history pages by trade ID, which relies on UUIDv7 IDs being time ordered
	id, idErr := uuid.NewV7()
	if idErr != nil {
		return pkgerrors.Wrap(idErr, "gen trade id")
	}
	trade.ID = id

	notional, notionalErr := trade.Notional()
	if notionalErr != nil {
		return pkgerrors.Wrap(notionalErr, "trade notional")
	}

	fn := func(ctx context.Context) error {
		if err := s.tradeRepo.CreateTrade(ctx, trade); err != nil {
			return pkgerrors.Wrap(err, "trade repo create trade")
		}

		// The buyer pays the quote asset, the seller pays the base asset. Both were
		// reserved when the orders were created.
		if err := s.walletRepo.TransferFunds(
			ctx,
			trade.Bid.UserID,
			trade.Ask.UserID,
			trade.MarketPair.Quote.ID,
			notional,
		); err != nil {
			return pkgerrors.Wrap(err, "wallet repo transfer quote funds")
		}
		if err := s.walletRepo.TransferFunds(
			ctx,
			trade.Ask.UserID,
			trade.Bid.UserID,
			trade.MarketPair.Base.ID,
			trade.Quantity,
		); err != nil {
			return pkgerrors.Wrap(err, "wallet repo transfer base funds")
		}

		if err := s.releaseExcessReservation(ctx, trade); err != nil {
			return pkgerrors.Wrap(err, "release excess reservation")
		}

		if err := s.orderRepo.FillOrder(ctx, trade.Bid.OrderID, trade.Quantity); err != nil {
			return pkgerrors.Wrap(err, "order repo fill bid order")
		}
		if err := s.orderRepo.FillOrder(ctx, trade.Ask.OrderID, trade.Quantity); err != nil {
			return pkgerrors.Wrap(err, "order repo fill ask order")
		}

		return nil
	}

	if err := s.txManager.WithTX(ctx, fn); err != nil {
		return pkgerrors.Wrap(err, "tx manager with tx")
	}

	return nil
}

// releaseExcessReservation returns the funds the buyer reserved on top of what the
// trade actually cost, which would stay locked otherwise.
func (s *SettlementService) releaseExcessReservation(ctx context.Context, trade entity.Trade) error {
	excess, err := trade.GetExcessReservedAmount()
	if err != nil {
		return pkgerrors.Wrap(err, "trade get excess reserved amount")
	}
	if excess <= 0 {
		return nil
	}

	if err = s.walletRepo.ReleaseReservedFunds(
		ctx,
		trade.Bid.UserID,
		trade.MarketPair.Quote.ID,
		excess,
	); err != nil {
		return pkgerrors.Wrap(err, "wallet repo release reserved funds")
	}

	return nil
}

func NewSettlementService(
	eventBus EventBus,
	tradeRepo TradeRepository,
	orderRepo OrderRepository,
	walletRepo WalletRepository,
	marketRepo MarketRepository,
	txManager TXManager,
	logger pkg.Logger,
) *SettlementService {
	return &SettlementService{
		eventBus:   eventBus,
		tradeRepo:  tradeRepo,
		orderRepo:  orderRepo,
		walletRepo: walletRepo,
		marketRepo: marketRepo,
		txManager:  txManager,
		logger:     logger,
	}
}
