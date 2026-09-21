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
	GetOrder(ctx context.Context, orderID uuid.UUID) (entity.Order, error)
	FillOrder(ctx context.Context, orderID uuid.UUID, quantity int64) error
}

type WalletRepository interface {
	TransferFunds(ctx context.Context, fromUserID, toUserID uuid.UUID, symbolID int, amount int64) error
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type TradeService struct {
	tradeWriter TradeWriter
	tradeRepo   TradeRepository
	orderRepo   OrderRepository
	walletRepo  WalletRepository
	marketRepo  MarketRepository
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
	id, idErr := uuid.NewV7()
	if idErr != nil {
		return pkgerrors.Wrap(idErr, "gen trade id")
	}
	trade.ID = id

	fn := func(ctx context.Context) error {
		bidOrder, err := s.orderRepo.GetOrder(ctx, trade.BidOrderID)
		if err != nil {
			return pkgerrors.Wrap(err, "order repo get bid order")
		}
		askOrder, err := s.orderRepo.GetOrder(ctx, trade.AskOrderID)
		if err != nil {
			return pkgerrors.Wrap(err, "order repo get ask order")
		}

		pair, ok := s.marketRepo.GetPair(bidOrder.MarketPair.ID)
		if !ok {
			return pkgerrors.Errorf("market repo pair %d doesn't exist", bidOrder.MarketPair.ID)
		}

		if err = s.tradeRepo.CreateTrade(ctx, trade); err != nil {
			return pkgerrors.Wrap(err, "trade repo create trade")
		}

		// The buyer pays the quote asset, the seller pays the base asset. Both were
		// reserved when the orders were created.
		if err = s.walletRepo.TransferFunds(
			ctx,
			bidOrder.UserID,
			askOrder.UserID,
			pair.Quote.ID,
			trade.Price*trade.Quantity,
		); err != nil {
			return pkgerrors.Wrap(err, "wallet repo transfer quote funds")
		}
		if err = s.walletRepo.TransferFunds(
			ctx,
			askOrder.UserID,
			bidOrder.UserID,
			pair.Base.ID,
			trade.Quantity,
		); err != nil {
			return pkgerrors.Wrap(err, "wallet repo transfer base funds")
		}

		if err = s.orderRepo.FillOrder(ctx, trade.BidOrderID, trade.Quantity); err != nil {
			return pkgerrors.Wrap(err, "order repo fill bid order")
		}
		if err = s.orderRepo.FillOrder(ctx, trade.AskOrderID, trade.Quantity); err != nil {
			return pkgerrors.Wrap(err, "order repo fill ask order")
		}

		return nil
	}

	if err := s.txManager.WithTX(ctx, fn); err != nil {
		return pkgerrors.Wrap(err, "tx manager with tx")
	}

	return nil
}

func NewTradeService(
	tradeWriter TradeWriter,
	tradeRepo TradeRepository,
	orderRepo OrderRepository,
	walletRepo WalletRepository,
	marketRepo MarketRepository,
	txManager TXManager,
	logger pkg.Logger,
) *TradeService {
	return &TradeService{
		tradeWriter: tradeWriter,
		tradeRepo:   tradeRepo,
		orderRepo:   orderRepo,
		walletRepo:  walletRepo,
		marketRepo:  marketRepo,
		txManager:   txManager,
		logger:      logger,
	}
}
