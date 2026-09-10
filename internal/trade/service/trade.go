package service

import (
	"context"
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
)

type TradeWriter interface {
	RetrieveTrades(ctx context.Context) ([]entity.Trade, error)
}

type TradeRepository interface {
	CreateTrades(ctx context.Context, trades []entity.Trade) error
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type TradeService struct {
	tradeWriter TradeWriter
	tradeRepo   TradeRepository
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

		idErr := assignTradeIDs(trades)
		if idErr != nil {
			s.logger.Errorf("gen trade id: %v", idErr)
			continue
		}

		fn := func(ctx context.Context) error {
			return s.tradeRepo.CreateTrades(ctx, trades)
		}

		if err = s.txManager.WithTX(ctx, fn); err != nil {
			s.logger.Errorf("tx manager with tx: %v", err)
			continue
		}
	}
}

func assignTradeIDs(trades []entity.Trade) error {
	for i := range trades {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		trades[i].ID = id
	}
	return nil
}

func NewTradeService(
	tradeWriter TradeWriter,
	tradeRepo TradeRepository,
	txManager TXManager,
	logger pkg.Logger,
) *TradeService {
	return &TradeService{
		tradeWriter: tradeWriter,
		tradeRepo:   tradeRepo,
		txManager:   txManager,
		logger:      logger,
	}
}
