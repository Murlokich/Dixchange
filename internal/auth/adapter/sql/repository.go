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
	emailUniqueConstraintName = "user_email_key"
)

var (
	ErrEmailIsAlreadyUsed = errors.New("email is already used")
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateUser(ctx context.Context, user entity.User) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateUser(ctx, CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == emailUniqueConstraintName {
			return errors.Wrap(ErrEmailIsAlreadyUsed, "query create user")
		}
		return errors.Wrap(err, "query create user")
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, errors.Wrap(entity.ErrNoUserFound, "query get user email")
		}
		return entity.User{}, errors.Wrap(err, "query get user email")
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
