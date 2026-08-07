package sql

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	noUserAffected = 0
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateBalanceRecord(ctx context.Context, balanceRecord entity.BalanceRecord) error {
	const errWrapMessage = "main repository CreateBalanceRecord"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateBalanceRecord(ctx, CreateBalanceRecordParams{
		ID:     balanceRecord.ID,
		UserID: balanceRecord.UserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(balanceRecord.SymbolID),
	}); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}

	return nil
}

func (r *Repository) Deposit(ctx context.Context, userID uuid.UUID, symbolID int, amount decimal.Decimal) error {
	const errWrapMessage = "main Repository DepositUSD"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.Deposit(ctx, DepositParams{
		Main:   amount,
		UserID: userID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(symbolID),
	})
	if rows == noUserAffected {
		return errors.Errorf("balance record of symbol: %d for the user doesn't exist", symbolID)
	}
	if err != nil {
		return errors.Wrap(err, errWrapMessage)
	}
	return nil
}

func (r *Repository) ReserveFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount decimal.Decimal) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.ReserveFunds(ctx, ReserveFundsParams{
		Main:   amount,
		UserID: userID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(symbolID),
	})
	if err != nil {
		return errors.Wrap(err, "query reserve funds")
	}
	if rows == noUserAffected {
		return entity.ErrInsufficientBalance
	}
	return nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
