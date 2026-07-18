package register

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user entity.User) error
}

type PasswordHasher interface {
	GeneratePasswordHash(password string) (string, error)
}

type BalanceRepository interface {
	CreateBalanceRecord(ctx context.Context, userID uuid.UUID) error
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type UseCase struct {
	userRepo    UserRepository
	pwdHasher   PasswordHasher
	balanceRepo BalanceRepository
	txManager   TXManager
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	const errWrapMessage = "register UseCase Execute"

	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}

	user, err := u.inputToUser(input)
	if err != nil {
		return Output{}, errors.Wrap(err, errWrapMessage)
	}

	fn := func(ctx context.Context) error {
		if err = u.userRepo.CreateUser(ctx, user); err != nil {
			return errors.Wrap(err, errWrapMessage)
		}

		if err = u.balanceRepo.CreateBalanceRecord(ctx, user.ID); err != nil {
			return errors.Wrap(err, errWrapMessage)
		}

		return nil
	}

	if err = u.txManager.WithTX(ctx, fn); err != nil {
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

func NewUseCase(
	userRepo UserRepository,
	pwdHasher PasswordHasher,
	balanceRepo BalanceRepository,
	txManager TXManager,
) *UseCase {
	return &UseCase{
		userRepo:    userRepo,
		pwdHasher:   pwdHasher,
		balanceRepo: balanceRepo,
		txManager:   txManager,
	}
}
