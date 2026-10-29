package register_test

import (
	"context"
	"testing"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/register"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	validEmail    = "user@example.com"
	validPassword = "Passw0rd!"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	userRepo := register.NewMockUserRepository(t)
	pwdHasher := register.NewMockPasswordHasher(t)
	balanceRepo := register.NewMockBalanceRepository(t)
	marketRepo := register.NewMockMarketRepository(t)
	txManager := register.NewMockTXManager(t)

	const hashedPassword = "hashed-password"
	defaultSymbols := []entity.Symbol{
		{ID: 1, Code: "USD"},
		{ID: 2, Code: "KCN"},
	}

	var capturedUser entity.User

	pwdHasher.EXPECT().GeneratePasswordHash(validPassword).Return(hashedPassword, nil).Once()

	userRepo.EXPECT().
		CreateUser(mock.Anything, mock.MatchedBy(func(user entity.User) bool {
			capturedUser = user
			return user.Email == validEmail && user.PasswordHash == hashedPassword && user.ID != uuid.Nil
		})).
		Return(nil).
		Once()

	marketRepo.EXPECT().GetDefaultSymbols().Return(defaultSymbols).Once()

	for _, symbol := range defaultSymbols {
		symbolID := symbol.ID
		balanceRepo.EXPECT().
			CreateBalanceRecord(mock.Anything, mock.MatchedBy(func(record entity.BalanceRecord) bool {
				return record.SymbolID == symbolID && record.UserID == capturedUser.ID && record.ID != uuid.Nil
			})).
			Return(nil).
			Once()
	}

	txManager.EXPECT().
		WithTX(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Once()

	uc := register.NewUseCase(userRepo, pwdHasher, balanceRepo, marketRepo, txManager)
	output, err := uc.Execute(t.Context(), register.Input{Email: validEmail, Password: validPassword})

	require.NoError(t, err)
	require.Equal(t, register.Output{}, output)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		email string
		pwd   string
	}{
		{name: "invalid email", email: "not-an-email", pwd: validPassword},
		{name: "invalid password", email: validEmail, pwd: "weak"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			userRepo := register.NewMockUserRepository(t)
			pwdHasher := register.NewMockPasswordHasher(t)
			balanceRepo := register.NewMockBalanceRepository(t)
			marketRepo := register.NewMockMarketRepository(t)
			txManager := register.NewMockTXManager(t)

			uc := register.NewUseCase(userRepo, pwdHasher, balanceRepo, marketRepo, txManager)
			_, err := uc.Execute(t.Context(), register.Input{Email: tc.email, Password: tc.pwd})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}
