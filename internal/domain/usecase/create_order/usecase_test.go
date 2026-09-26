package createorder_test

import (
	"context"
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	bookWriter := createorder.NewMockBookWriter(t)

	pair := entity.Pair{
		ID:             validMarketPairID,
		Base:           entity.Symbol{ID: 1, Code: "KCN", Precision: 0},
		Quote:          entity.Symbol{ID: 2, Code: "USD", Precision: 2},
		PricePrecision: 2,
	}
	// buy debits the quote asset; base precision + price precision == quote precision,
	// so the notional is plain price times quantity
	wantTotalPrice := validPricePerUnit * validQuantityOrdered

	marketRepo.EXPECT().GetPair(validMarketPairID).Return(pair, true).Once()

	// the conversion math itself is covered by the converter's own tests; here we just
	// pass the quantitative fields through so the rest of the usecase can be exercised.
	converter.EXPECT().
		DeriveOrderWithQuantitativeFields(mock.Anything, validQuantityDecimal, validPriceDecimal).
		RunAndReturn(func(order entity.Order, _ decimal.Decimal, _ decimal.Decimal) (entity.Order, error) {
			order.QuantityOrdered = validQuantityOrdered
			order.PricePerUnit = validPricePerUnit
			return order, nil
		}).
		Once()

	walletRepo.EXPECT().
		ReserveFunds(mock.Anything, validUUID, pair.Quote.ID, wantTotalPrice).
		Return(nil).
		Once()

	matchesExpectedOrder := func(order entity.Order) bool {
		return order.ID != uuid.Nil &&
			order.UserID == validUUID &&
			order.MarketPair == pair &&
			order.Side == entity.BuySide &&
			order.QuantityOrdered == validQuantityOrdered &&
			order.PricePerUnit == validPricePerUnit &&
			order.Status == entity.StatusOpen
	}

	orderRepo.EXPECT().
		CreateOrder(mock.Anything, mock.MatchedBy(matchesExpectedOrder)).
		Return(nil).
		Once()

	bookWriter.EXPECT().
		InsertOrder(mock.Anything, mock.MatchedBy(matchesExpectedOrder)).
		Return(nil).
		Once()

	txManager.EXPECT().
		WithTX(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, bookWriter)
	output, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Action:          entity.BuySide,
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
		{name: "nil user id", userID: uuid.Nil, action: entity.BuySide},
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
			converter := createorder.NewMockConverter(t)
			bookWriter := createorder.NewMockBookWriter(t)

			uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, bookWriter)
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
	converter := createorder.NewMockConverter(t)
	bookWriter := createorder.NewMockBookWriter(t)

	marketRepo.EXPECT().GetPair(validMarketPairID).Return(entity.Pair{}, false).Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, bookWriter)
	_, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Action:          entity.BuySide,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo pair doesn't exist")
}
