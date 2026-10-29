package gettrades

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type TradeRepository interface {
	GetUserTrades(ctx context.Context, userID uuid.UUID, cursor *uuid.UUID, limit int) ([]entity.Trade, error)
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
}

type UseCase struct {
	tradeRepo  TradeRepository
	marketRepo MarketRepository
	pageSize   int
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	// one extra trade tells whether another page exists without querying for it
	trades, err := u.tradeRepo.GetUserTrades(ctx, input.UserID, input.Cursor, u.pageSize+1)
	if err != nil {
		return Output{}, errors.Wrap(err, "trade repo get user trades")
	}
	hasMore := len(trades) > u.pageSize
	if hasMore {
		trades = trades[:u.pageSize]
	}

	output := Output{Trades: make([]UserTrade, 0, len(trades))}
	for i := range trades {
		userTrade, tradeErr := u.toUserTrade(trades[i], input.UserID)
		if tradeErr != nil {
			return Output{}, errors.Wrapf(tradeErr, "trade %s", trades[i].ID)
		}
		output.Trades = append(output.Trades, userTrade)
	}

	if hasMore {
		output.NextCursor = &trades[len(trades)-1].ID
	}
	return output, nil
}

// toUserTrade resolves the market pair of the trade, which the repository returns
// as an ID only, and determines the part of the trade that belongs to the user.
func (u *UseCase) toUserTrade(trade entity.Trade, userID uuid.UUID) (UserTrade, error) {
	pair, ok := u.marketRepo.GetPair(trade.MarketPair.ID)
	if !ok {
		return UserTrade{}, errors.Errorf("market repo pair %d doesn't exist", trade.MarketPair.ID)
	}
	trade.MarketPair = pair

	party, err := trade.PartyByUserID(userID)
	if err != nil {
		return UserTrade{}, errors.Wrap(err, "trade party of")
	}

	total, err := trade.Notional()
	if err != nil {
		return UserTrade{}, errors.Wrap(err, "trade notional")
	}

	return UserTrade{Trade: trade, Party: party, Total: total}, nil
}

// NewUseCase creates the usecase; pageSize must be positive.
func NewUseCase(tradeRepo TradeRepository, marketRepo MarketRepository, pageSize int) *UseCase {
	return &UseCase{tradeRepo: tradeRepo, marketRepo: marketRepo, pageSize: pageSize}
}
