package deposit_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/dixchange/Monolith/internal/market/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	walletRepo := deposit.NewMockWalletRepository(t)
	marketRepo := deposit.NewMockMarketRepository(t)
	converter := deposit.NewMockConverter(t)

	marketRepo.EXPECT().GetSymbol(validSymbolID).Return(entity.Symbol{}, true).Once()
	converter.EXPECT().ConvertAmountToInt(entity.Symbol{}, validAmountDecimal).Return(int64(validAmount), nil).Once()
	walletRepo.EXPECT().Deposit(mock.Anything, validUUID, validSymbolID, int64(validAmount)).Return(nil).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo, converter)
	output, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   validAmountDecimal,
	})

	require.NoError(t, err)
	require.Equal(t, deposit.Output{}, output)
}

// TestUseCase_Execute_ConvertsAmountUsingSymbolPrecision uses the real converter (not a mock) to
// check that a decimal amount is actually scaled to the symbol's precision before it reaches the
// wallet repository, rather than just checking the usecase forwards whatever the converter
// happens to return.
func TestUseCase_Execute_ConvertsAmountUsingSymbolPrecision(t *testing.T) {
	t.Parallel()

	symbol := entity.Symbol{ID: validSymbolID, Code: "KCN", Precision: 2}
	amount := decimal.RequireFromString("10.56") // exactly matches the symbol's precision
	const expectedAmount = int64(1056)           // shifted by 2

	walletRepo := deposit.NewMockWalletRepository(t)
	marketRepo := deposit.NewMockMarketRepository(t)

	marketRepo.EXPECT().GetSymbol(validSymbolID).Return(symbol, true).Once()
	walletRepo.EXPECT().Deposit(mock.Anything, validUUID, validSymbolID, expectedAmount).Return(nil).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo, &service.Converter{})
	_, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   amount,
	})

	require.NoError(t, err)
}

// TestUseCase_Execute_RejectsAmountWithExcessPrecision uses the real converter to check that an
// amount with more decimal places than the symbol supports is rejected rather than silently
// truncated, and that the wallet repository is never reached in that case.
func TestUseCase_Execute_RejectsAmountWithExcessPrecision(t *testing.T) {
	t.Parallel()

	symbol := entity.Symbol{ID: validSymbolID, Code: "KCN", Precision: 2}
	amount := decimal.RequireFromString("10.567") // one digit past the symbol's precision

	walletRepo := deposit.NewMockWalletRepository(t)
	marketRepo := deposit.NewMockMarketRepository(t)

	marketRepo.EXPECT().GetSymbol(validSymbolID).Return(symbol, true).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo, &service.Converter{})
	_, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   amount,
	})

	require.ErrorContains(t, err, "more decimal places")
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
			converter := deposit.NewMockConverter(t)

			uc := deposit.NewUseCase(walletRepo, marketRepo, converter)
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
	converter := deposit.NewMockConverter(t)

	marketRepo.EXPECT().GetSymbol(validSymbolID).Return(entity.Symbol{}, false).Once()

	uc := deposit.NewUseCase(walletRepo, marketRepo, converter)
	_, err := uc.Execute(t.Context(), deposit.Input{
		UserID:   validUUID,
		SymbolID: validSymbolID,
		Amount:   validAmountDecimal,
	})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo symbol doesn't exist")
}
