package createorder_test

import (
	"context"
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)

	pair := entity.Pair{
		ID:    validMarketPairID,
		Base:  entity.Symbol{ID: 1, Code: "KCN"},
		Quote: entity.Symbol{ID: 2, Code: "USD"},
	}
	// buy debits the quote asset
	wantTotalPrice := validPriceDecimal.Mul(validQuantityDecimal)

	marketRepo.EXPECT().PairExists(validMarketPairID).Return(true).Once()
	marketRepo.EXPECT().GetPair(validMarketPairID).Return(pair).Once()

	walletRepo.EXPECT().
		ReserveFunds(mock.Anything, validUUID, pair.Quote.ID, wantTotalPrice).
		Return(nil).
		Once()

	orderRepo.EXPECT().
		CreateOrder(mock.Anything, mock.MatchedBy(func(order entity.Order) bool {
			return order.ID != uuid.Nil &&
				order.UserID == validUUID &&
				order.MarketPair == pair &&
				order.Action == entity.BuyAction &&
				order.QuantityOrdered.Equal(validQuantityDecimal) &&
				order.PricePerUnit.Equal(validPriceDecimal) &&
				order.Status == entity.StatusOpen
		})).
		Return(nil).
		Once()

	txManager.EXPECT().
		WithTX(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager)
	output, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Action:          entity.BuyAction,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	})

	require.NoError(t, err)
	require.Equal(t, createorder.Output{}, output)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		userID uuid.UUID
		action string
	}{
		{name: "nil user id", userID: uuid.Nil, action: entity.BuyAction},
		{name: "invalid action", userID: validUUID, action: "gibberish"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			orderRepo := createorder.NewMockOrderRepository(t)
			walletRepo := createorder.NewMockWalletRepository(t)
			marketRepo := createorder.NewMockMarketRepository(t)
			txManager := createorder.NewMockTXManager(t)

			uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager)
			_, err := uc.Execute(t.Context(), createorder.Input{
				UserID:          tc.userID,
				MarketPairID:    validMarketPairID,
				Action:          tc.action,
				QuantityOrdered: validQuantityDecimal,
				PricePerUnit:    validPriceDecimal,
			})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_PairDoesNotExist(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)

	marketRepo.EXPECT().PairExists(validMarketPairID).Return(false).Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager)
	_, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Action:          entity.BuyAction,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo pair doesn't exist")
}
