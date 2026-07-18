package deposit

import (
	"context"

	"github.com/google/uuid"
	decimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/pkg/errors"
)

type Repository interface {
	DepositUSD(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) error
}

type UseCase struct {
	repo Repository
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	const errWrapMessage = "deposit UseCase Execute"

	if err := u.repo.DepositUSD(ctx, input.UserID, input.USDAmount); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}
	return Output{}, nil
}
