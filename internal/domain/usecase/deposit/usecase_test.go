package deposit_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	walletRepo := deposit.NewMockWalletRepository(t)
	marketRepo := deposit.NewMockMarketRepository(t)

	marketRepo.EXPECT().SymbolExists(validSymbolID).Return(true).Once()
	walletRepo.EXPECT().Deposit(mock.Anything, validUUID, validSymbolID, validAmountDecimal).Return(nil).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo)
	output, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   validAmountDecimal,
	})

	require.NoError(t, err)
	require.Equal(t, deposit.Output{}, output)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		userID   uuid.UUID
		symbolID int
		amount   decimal.Decimal
	}{
		{name: "nil user id", userID: uuid.Nil, symbolID: validSymbolID, amount: validAmountDecimal},
		{name: "negative symbol id", userID: validUUID, symbolID: -1, amount: validAmountDecimal},
		{name: "non-positive amount", userID: validUUID, symbolID: validSymbolID, amount: decimal.Zero},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			walletRepo := deposit.NewMockWalletRepository(t)
			marketRepo := deposit.NewMockMarketRepository(t)

			uc := deposit.NewUseCase(walletRepo, marketRepo)
			_, err := uc.Execute(t.Context(), deposit.Input{
				UserID:   tc.userID,
				SymbolID: tc.symbolID,
				Amount:   tc.amount,
			})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_SymbolDoesNotExist(t *testing.T) {
	t.Parallel()

	walletRepo := deposit.NewMockWalletRepository(t)
	marketRepo := deposit.NewMockMarketRepository(t)

	marketRepo.EXPECT().SymbolExists(validSymbolID).Return(false).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo)
	_, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   validAmountDecimal,
	})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo symbol doesn't exist")
}
