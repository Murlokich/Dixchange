package service_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/market/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestConverter_ConvertAmountToInt(t *testing.T) {
	t.Parallel()

	c := service.NewConverter()

	testCases := []struct {
		name        string
		symbol      entity.Symbol
		amount      decimal.Decimal
		expected    int64
		errContains string
	}{
		{
			name:     "happy path: shifts by precision",
			symbol:   entity.Symbol{Code: "BTC", Precision: 8},
			amount:   decimal.RequireFromString("1.2345"),
			expected: 123450000,
		},
		{
			name:     "happy path: amount uses exactly all supported decimal places",
			symbol:   entity.Symbol{Code: "USD", Precision: 2},
			amount:   decimal.RequireFromString("10.50"),
			expected: 1050,
		},
		{
			name:     "precision zero: whole number passes through unchanged",
			symbol:   entity.Symbol{Code: "KCN", Precision: 0},
			amount:   decimal.RequireFromString("1000"),
			expected: 1000,
		},
		{
			name:        "precision zero: fractional amount rejected",
			symbol:      entity.Symbol{Code: "KCN", Precision: 0},
			amount:      decimal.RequireFromString("1000.5"),
			errContains: "more decimal places",
		},
		{
			name:        "excess decimal places beyond precision rejected",
			symbol:      entity.Symbol{Code: "BTC", Precision: 8},
			amount:      decimal.RequireFromString("1.234567891"),
			errContains: "more decimal places",
		},
		{
			name:        "shifted amount exceeds int64 range",
			symbol:      entity.Symbol{Code: "HUGE", Precision: 18},
			amount:      decimal.RequireFromString("10"),
			errContains: "exceeds int64 range",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := c.ConvertAmountToInt(tc.symbol, tc.amount)
			if tc.errContains != "" {
				require.ErrorContains(t, err, tc.errContains)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestConverter_DeriveOrderWithQuantitativeFields_HappyPath(t *testing.T) {
	t.Parallel()

	c := service.NewConverter()
	order := baseOrder()

	got, err := c.DeriveOrderWithQuantitativeFields(
		order,
		decimal.RequireFromString("0.75"),
		decimal.RequireFromString("65432.10"),
	)

	require.NoError(t, err)

	expected := order
	expected.QuantityOrdered = 75000000 // 0.75 BTC shifted by base precision 8
	expected.PricePerUnit = 6543210     // 65432.10 USD shifted by quote precision 2
	require.Equal(t, expected, got)
}

func TestConverter_DeriveOrderWithQuantitativeFields_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		quantity    decimal.Decimal
		price       decimal.Decimal
		errIs       error
		errContains string
	}{
		{
			name:        "quantity has more decimal places than base precision",
			quantity:    decimal.RequireFromString("0.123456789"), // 9 decimals, base precision 8
			price:       decimal.RequireFromString("65432.10"),
			errContains: "more decimal places",
		},
		{
			name:        "price has more decimal places than quote precision",
			quantity:    decimal.RequireFromString("0.75"),
			price:       decimal.RequireFromString("65432.109"), // 3 decimals, quote precision 2
			errContains: "more decimal places",
		},
		{
			name:     "quantity is not a multiple of the pair's lot size",
			quantity: decimal.RequireFromString("0.00000999"), // 999 satoshis, lot size 1000
			price:    decimal.RequireFromString("65432.10"),
			errIs:    service.ErrInvalidNumberOfLots,
		},
		{
			name:     "price is not a multiple of the pair's price tick",
			quantity: decimal.RequireFromString("0.75"),
			price:    decimal.RequireFromString("65432.12"), // 6543212 cents, tick 5
			errIs:    service.ErrInvalidNumberOfPriceTicks,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := service.NewConverter()
			got, err := c.DeriveOrderWithQuantitativeFields(baseOrder(), tc.quantity, tc.price)

			require.Error(t, err)
			require.Equal(t, entity.Order{}, got)

			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
			} else {
				require.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}

func validPair() entity.Pair {
	return entity.Pair{
		Base:      entity.Symbol{Code: "BTC", Precision: 8},
		Quote:     entity.Symbol{Code: "USD", Precision: 2},
		LotSize:   1000, // 0.00001 BTC
		PriceTick: 5,    // 5 cents
	}
}

func baseOrder() entity.Order {
	return entity.Order{
		ID:         uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e33"),
		UserID:     uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e44"),
		MarketPair: validPair(),
		Action:     entity.BuyAction,
		Status:     entity.StatusOpen,
	}
}
