package register

import (
	"context"

	"github.com/dixchange/Monolith/internal/auth/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrEmailIsAlreadyUsed = errors.New("email is already used")
)

type PasswordHasher interface {
	GeneratePasswordHash(password string) (string, error)
}

type Repository interface {
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	CreateUser(ctx context.Context, user entity.User) error
}

type UseCase struct {
	repo      Repository
	pwdHasher PasswordHasher
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	const errWrapMessage = "register UseCase execute"
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}

	exists, err := u.repo.IsEmailTaken(ctx, input.Email)
	if err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}
	if exists {
		// don't register, but don't explain to user to prevent user enumeration
		// for now just return error - user exists
		return Output{}, errors.Wrap(ErrEmailIsAlreadyUsed, errWrapMessage)
	}
	user, err := u.inputToUser(input)
	if err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}
	if err = u.repo.CreateUser(ctx, user); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}
	return Output{}, nil
}

func (u *UseCase) inputToUser(input Input) (entity.User, error) {
	const errWrapMessage = "register inputToUser"
	id, err := uuid.NewV7()
	if err != nil {
		return entity.User{}, errors.Wrap(err, errWrapMessage)
	}
	pwdHash, err := u.pwdHasher.GeneratePasswordHash(input.Password)
	if err != nil {
		return entity.User{}, errors.Wrap(err, errWrapMessage)
	}

	return entity.User{
		ID:           id,
		Email:        input.Email,
		PasswordHash: pwdHash,
	}, nil
}

func NewUseCase(repo Repository, pwdHasher PasswordHasher) *UseCase {
	return &UseCase{
		repo:      repo,
		pwdHasher: pwdHasher,
	}
}
