package deposit

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Repository interface {
	Deposit(ctx context.Context, userID uuid.UUID, symbolID int, amount decimal.Decimal) error
}

type UseCase struct {
	repo Repository
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	const errWrapMessage = "deposit UseCase Execute"

	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}

	if err := u.repo.Deposit(ctx, input.UserID, input.SymbolID, input.Amount); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}
	return Output{}, nil
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}
