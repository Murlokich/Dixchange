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
	CreateBalanceRecord(ctx context.Context, balanceRecord entity.BalanceRecord) error
}

type MarketRepository interface {
	GetDefaultSymbols() []entity.Symbol
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type UseCase struct {
	userRepo    UserRepository
	pwdHasher   PasswordHasher
	balanceRepo BalanceRepository
	marketRepo  MarketRepository
	txManager   TXManager
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	user, err := u.inputToUser(input)
	if err != nil {
		return Output{}, errors.Wrap(err, "input to user")
	}

	fn := func(ctx context.Context) error {
		if err = u.userRepo.CreateUser(ctx, user); err != nil {
			return errors.Wrap(err, "user repo create user")
		}

		symbols := u.marketRepo.GetDefaultSymbols()

		for i := range symbols {
			var recordID uuid.UUID
			recordID, err = uuid.NewV7()
			if err != nil {
				return errors.Wrap(err, "gen record uuid")
			}
			if err = u.balanceRepo.CreateBalanceRecord(ctx, entity.BalanceRecord{
				ID:       recordID,
				UserID:   user.ID,
				SymbolID: symbols[i].ID,
			}); err != nil {
				return errors.Wrap(err, "balance repo create record")
			}
		}

		return nil
	}

	if err = u.txManager.WithTX(ctx, fn); err != nil {
		return Output{}, errors.Wrap(err, "tx manager with tx")
	}

	return Output{}, nil
}

func (u *UseCase) inputToUser(input Input) (entity.User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return entity.User{}, errors.Wrap(err, "uuid new")
	}
	pwdHash, err := u.pwdHasher.GeneratePasswordHash(input.Password)
	if err != nil {
		return entity.User{}, errors.Wrap(err, "password hasher gen hash")
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
	marketRepo MarketRepository,
	txManager TXManager,
) *UseCase {
	return &UseCase{
		userRepo:    userRepo,
		pwdHasher:   pwdHasher,
		balanceRepo: balanceRepo,
		marketRepo:  marketRepo,
		txManager:   txManager,
	}
}
