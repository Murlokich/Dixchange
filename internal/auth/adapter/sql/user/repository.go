package user

import (
	"context"

	"github.com/pkg/errors"

	"github.com/dixchange/Monolith/internal/auth/domain/entity"
)

type Repository struct {
	q *Queries
}

func (r *Repository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	const errWrapMessage = "user repository IsEmailTaken"
	res, err := r.q.IsEmailTaken(ctx, email)
	return res, errors.Wrap(err, errWrapMessage)
}

func (r *Repository) CreateUser(ctx context.Context, user entity.User) error {
	const errWrapMessage = "user repository CreateUser"
	if err := r.q.CreateUser(ctx, CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	const errWrapMessage = "user repository GetUserByEmail"
	user, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
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
