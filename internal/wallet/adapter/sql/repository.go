package sql

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	noUserAffected = 0
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateBalanceRecord(ctx context.Context, balanceRecord entity.BalanceRecord) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateBalanceRecord(ctx, CreateBalanceRecordParams{
		ID:     balanceRecord.ID,
		UserID: balanceRecord.UserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(balanceRecord.SymbolID),
	}); err != nil {
		return errors.Wrap(err, "query create balance record")
	}

	return nil
}

// GetPairBalances returns the user balances of the pair base and quote symbols in a single query,
// so both are read from the same snapshot. A symbol without a record is omitted.
func (r *Repository) GetPairBalances(
	ctx context.Context,
	userID uuid.UUID,
	baseSymbolID int,
	quoteSymbolID int,
) ([]entity.Balance, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	balances, err := q.GetPairBalances(ctx, GetPairBalancesParams{
		UserID: userID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(baseSymbolID),
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID_2: int32(quoteSymbolID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "query get pair balances")
	}

	result := make([]entity.Balance, 0, len(balances))
	for _, b := range balances {
		result = append(result, entity.Balance{
			SymbolID: int(b.SymbolID),
			Main:     b.Main,
			Reserved: b.Reserved,
		})
	}
	return result, nil
}

func (r *Repository) Deposit(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.Deposit(ctx, DepositParams{
		Main:   amount,
		UserID: userID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(symbolID),
	})
	if err != nil {
		return errors.Wrap(err, "query deposit")
	}
	if rows == noUserAffected {
		return errors.Errorf("balance record of symbol: %d for the user doesn't exist", symbolID)
	}
	return nil
}

func (r *Repository) ReserveFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error {
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

// ReleaseReservedFunds moves amount of the symbol from the user reserved funds back
// to their main funds. It is the inverse of ReserveFunds.
func (r *Repository) ReleaseReservedFunds(
	ctx context.Context,
	userID uuid.UUID,
	symbolID int,
	amount int64,
) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.ReleaseReservedFunds(ctx, ReleaseReservedFundsParams{
		Reserved: amount,
		UserID:   userID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(symbolID),
	})
	if err != nil {
		return errors.Wrap(err, "query release reserved funds")
	}
	if rows == noUserAffected {
		return entity.ErrInsufficientBalance
	}

	return nil
}

// TransferFunds moves amount of the symbol from the payer reserved funds to the payee
// main funds. Both legs are applied by a single query, so the payee is credited only
// when the payer had enough funds reserved.
func (r *Repository) TransferFunds(
	ctx context.Context,
	fromUserID uuid.UUID,
	toUserID uuid.UUID,
	symbolID int,
	amount int64,
) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.TransferFunds(ctx, TransferFundsParams{
		Amount:     amount,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		SymbolID: int32(symbolID),
	})
	if err != nil {
		return errors.Wrap(err, "query transfer funds")
	}
	if rows == noUserAffected {
		return entity.ErrInsufficientBalance
	}

	return nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
