package sql

import (
	"context"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
)

const (
	emailColumnName = "email"
)

var (
	ErrEmailIsAlreadyUsed = errors.New("email is already used")
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateUser(ctx context.Context, user entity.User) error {
	const errWrapMessage = "user repository CreateUser"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateUser(ctx, CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ColumnName == emailColumnName {
			return errors.Wrap(ErrEmailIsAlreadyUsed, errWrapMessage)
		}
		return errors.Wrap(err, errWrapMessage)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	const errWrapMessage = "user repository GetUserByEmail"

	db := r.txGetter(ctx, r.db)
	q := New(db)

	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, errors.Wrap(entity.ErrNoUserFound, errWrapMessage)
		}
		return entity.User{}, errors.Wrap(err, errWrapMessage)
	}
	return entity.User{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}, nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
