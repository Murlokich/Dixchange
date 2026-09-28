package getpairbalance

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type WalletRepository interface {
	GetPairBalances(
		ctx context.Context,
		userID uuid.UUID,
		baseSymbolID int,
		quoteSymbolID int,
	) ([]entity.Balance, error)
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
}

type UseCase struct {
	walletRepo WalletRepository
	marketRepo MarketRepository
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	pair, ok := u.marketRepo.GetPair(input.PairID)
	if !ok {
		return Output{}, entity.ValidationError{Err: errors.New("market repo pair doesn't exist")}
	}

	balances, err := u.walletRepo.GetPairBalances(ctx, input.UserID, pair.Base.ID, pair.Quote.ID)
	if err != nil {
		return Output{}, errors.Wrap(err, "wallet repo get pair balances")
	}

	output := Output{Pair: pair}
	var hasBase, hasQuote bool
	for _, b := range balances {
		switch b.SymbolID {
		case pair.Base.ID:
			output.Base, hasBase = b, true
		case pair.Quote.ID:
			output.Quote, hasQuote = b, true
		}
	}
	if !hasBase || !hasQuote {
		return Output{}, errors.Wrapf(entity.ErrBalanceNotFound, "pair %d", pair.ID)
	}

	return output, nil
}

func NewUseCase(walletRepo WalletRepository, marketRepo MarketRepository) *UseCase {
	return &UseCase{walletRepo: walletRepo, marketRepo: marketRepo}
}
