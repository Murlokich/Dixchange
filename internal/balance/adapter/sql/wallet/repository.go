package wallet

import (
	"context"

	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateBalanceRecord(ctx context.Context, userID uuid.UUID) error {
	const errWrapMessage = "main repository CreateBalanceRecord"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateBalanceRecord(ctx, userID); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}

	return nil
}

func (r *Repository) DepositUSD(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) error {
	const errWrapMessage = "main Repository DepositUSD"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.DepositUSD(ctx, DepositUSDParams{
		Usd:    amount,
		UserID: userID,
	}); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}
	return nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
