package deposit

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type WalletRepository interface {
	Deposit(ctx context.Context, userID uuid.UUID, symbolID int, amount decimal.Decimal) error
}

type MarketRepository interface {
	SymbolExists(id int) bool
}

type UseCase struct {
	walletRepo WalletRepository
	marketRepo MarketRepository
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	if !u.marketRepo.SymbolExists(input.SymbolID) {
		return Output{}, entity.ValidationError{Err: errors.New("market repo symbol doesn't exist")}
	}

	if err := u.walletRepo.Deposit(ctx, input.UserID, input.SymbolID, input.Amount); err != nil {
		return Output{}, errors.Wrap(err, "wallet repo deposit")
	}
	return Output{}, nil
}

func NewUseCase(walletRepo WalletRepository, marketRepo MarketRepository) *UseCase {
	return &UseCase{walletRepo: walletRepo, marketRepo: marketRepo}
}
