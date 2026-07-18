package wallet

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Repository struct {
	q *Queries
}

func (r *Repository) CreateBalanceRecord(ctx context.Context, userID uuid.UUID) error {
	const errWrapMessage = "main repository CreateBalanceRecord"

	if err := r.q.CreateBalanceRecord(ctx, userID); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}

	return nil
}

func (r *Repository) DepositUSD(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) error {
	const errWrapMessage = "main Repository DepositUSD"

	if err := r.q.DepositUSD(ctx, DepositUSDParams{
		Usd:    amount,
		UserID: userID,
	}); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}
	return nil
}

func NewRepository(db DBTX) *Repository {
	return &Repository{q: New(db)}
}
