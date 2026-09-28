package service

import (
	"context"
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	pkgerrors "github.com/pkg/errors"
)

type TradeWriter interface {
	RetrieveTrades(ctx context.Context) ([]entity.Trade, error)
}

type TradeRepository interface {
	CreateTrade(ctx context.Context, trade entity.Trade) error
}

type OrderRepository interface {
	FillOrder(ctx context.Context, orderID uuid.UUID, quantity int64) error
}

type WalletRepository interface {
	TransferFunds(ctx context.Context, fromUserID, toUserID uuid.UUID, symbolID int, amount int64) error
	ReleaseReservedFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type TradeService struct {
	tradeWriter TradeWriter
	tradeRepo   TradeRepository
	orderRepo   OrderRepository
	walletRepo  WalletRepository
	txManager   TXManager
	logger      pkg.Logger
}

func (s *TradeService) Run(ctx context.Context) {
	for {
		trades, err := s.tradeWriter.RetrieveTrades(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			s.logger.Errorf("tradeWriter Retrieve trades: %v", err)
			continue
		}

		for i := range trades {
			if err = s.executeTrade(ctx, trades[i]); err != nil {
				if errors.Is(err, context.Canceled) {
					// TODO: graceful shutdown
					return
				}
				// TODO: think of the guarantee to actually execute trade or pesist it
				// halt on settlement failure — so the book stops diverging from the DB
				// log the failed trade and its error — so you can diagnose
				// reload open orders on restart — so the trade regenerates
				s.logger.Errorf("execute trade: %v", err)
			}
		}
	}
}

// executeTrade settles a single trade: it records the trade, moves the funds between
// the participants and updates the filled quantity of both orders. The whole
// settlement is applied within one transaction, so a trade is either fully settled
// or not settled at all.
func (s *TradeService) executeTrade(ctx context.Context, trade entity.Trade) error {
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
func (s *TradeService) releaseExcessReservation(ctx context.Context, trade entity.Trade) error {
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

func NewTradeService(
	tradeWriter TradeWriter,
	tradeRepo TradeRepository,
	orderRepo OrderRepository,
	walletRepo WalletRepository,
	txManager TXManager,
	logger pkg.Logger,
) *TradeService {
	return &TradeService{
		tradeWriter: tradeWriter,
		tradeRepo:   tradeRepo,
		orderRepo:   orderRepo,
		walletRepo:  walletRepo,
		txManager:   txManager,
		logger:      logger,
	}
}
