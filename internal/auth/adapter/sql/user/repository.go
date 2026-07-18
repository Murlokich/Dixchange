package user

import (
	"context"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

const (
	emailColumnName = "email"
)

var (
	ErrEmailIsAlreadyUsed = errors.New("email is already used")
)

type Repository struct {
	q *Queries
}

func (r *Repository) CreateUser(ctx context.Context, user entity.User) error {
	const errWrapMessage = "user repository CreateUser"

	if err := r.q.CreateUser(ctx, CreateUserParams{
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

	user, err := r.q.GetUserByEmail(ctx, email)
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

func NewRepository(db DBTX) *Repository {
	return &Repository{q: New(db)}
}
