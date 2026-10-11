package entity_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

var testPair = entity.Pair{ //nolint:gochecknoglobals // acceptable in test
	Base:  entity.Symbol{Code: "BTC"},
	Quote: entity.Symbol{Code: "USD"},
}

func TestOrder_GetDebitAsset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		side      string
		wantAsset entity.Symbol
		wantErr   bool
	}{
		{name: "buy", side: entity.BuySide, wantAsset: testPair.Quote},
		{name: "sell", side: entity.SellSide, wantAsset: testPair.Base},
		{name: "invalid side", side: "gibberish", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order := entity.Order{MarketPair: testPair, Side: tc.side}
			asset, err := order.GetDebitAsset()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAsset, asset)
		})
	}
}

func TestOrder_GetDebitOrderedAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		order       entity.Order
		want        int64
		errContains string
	}{
		{
			name: "buy debits notional in quote minor units: 0.00123 BTC at 65000.50 USDT",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Side:            entity.BuySide,
				QuantityOrdered: 123000,
				PricePerUnit:    6500050,
			},
			want: 7995061500,
		},
		{
			name: "buy debits notional in quote minor units: 1 USD at 1 KCN",
			order: entity.Order{
				MarketPair:      usdKcn,
				Side:            entity.BuySide,
				QuantityOrdered: 100,
				PricePerUnit:    1,
			},
			want: 1,
		},
		{
			name: "sell debits quantity in base minor units",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Side:            entity.SellSide,
				QuantityOrdered: 123000,
				PricePerUnit:    6500050,
			},
			want: 123000,
		},
		{
			name: "buy propagates pair notional error",
			order: entity.Order{
				MarketPair:      usdKcn,
				Side:            entity.BuySide,
				QuantityOrdered: 1,
				PricePerUnit:    1,
			},
			errContains: "market pair notional",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			amount, err := tc.order.GetDebitOrderedAmount()
			if tc.errContains != "" {
				require.ErrorContains(t, err, tc.errContains)
				require.Zero(t, amount)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, amount)
		})
	}
}

func TestOrder_GetDebitRemainingAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		order entity.Order
		want  int64
	}{
		{
			name: "buy debits notional of the unfilled part: 0.001 of 0.00123 BTC at 65000.50 USDT",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Side:            entity.BuySide,
				QuantityOrdered: 123000,
				QuantityFilled:  23000,
				PricePerUnit:    6500050,
			},
			want: 6500050000,
		},
		{
			name: "sell debits the unfilled quantity in base minor units",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Side:            entity.SellSide,
				QuantityOrdered: 123000,
				QuantityFilled:  23000,
				PricePerUnit:    6500050,
			},
			want: 100000,
		},
		{
			name: "unfilled order debits the ordered amount",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Side:            entity.BuySide,
				QuantityOrdered: 123000,
				PricePerUnit:    6500050,
			},
			want: 7995061500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			amount, err := tc.order.GetDebitRemainingAmount()
			require.NoError(t, err)
			require.Equal(t, tc.want, amount)
		})
	}
}

// What a cancel releases plus what the fills consumed must add up to what was reserved,
// otherwise funds are left locked in reserved forever.
func TestOrder_DebitAmountsAddUp(t *testing.T) {
	t.Parallel()

	order := entity.Order{
		MarketPair:      btcUsdt,
		Side:            entity.BuySide,
		QuantityOrdered: 123000,
		QuantityFilled:  23000,
		PricePerUnit:    6500050,
	}

	ordered, err := order.GetDebitOrderedAmount()
	require.NoError(t, err)
	remaining, err := order.GetDebitRemainingAmount()
	require.NoError(t, err)
	consumed, err := order.MarketPair.Notional(order.QuantityFilled, order.PricePerUnit)
	require.NoError(t, err)

	require.Equal(t, ordered, remaining+consumed)
}

func TestIsActiveOrderStatus(t *testing.T) {
	t.Parallel()

	require.True(t, entity.IsActiveOrderStatus(entity.StatusOpen))
	require.True(t, entity.IsActiveOrderStatus(entity.StatusPartiallyFilled))
	require.False(t, entity.IsActiveOrderStatus(entity.StatusFilled))
	require.False(t, entity.IsActiveOrderStatus(entity.StatusCancelled))
}

func TestOrder_GetCreditAsset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		side      string
		wantAsset entity.Symbol
		wantErr   bool
	}{
		{name: "buy", side: entity.BuySide, wantAsset: testPair.Base},
		{name: "sell", side: entity.SellSide, wantAsset: testPair.Quote},
		{name: "invalid side", side: "gibberish", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order := entity.Order{MarketPair: testPair, Side: tc.side}
			asset, err := order.GetCreditAsset()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAsset, asset)
		})
	}
}

func TestOrder_RemainingQuantity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		filled int64
		want   int64
	}{
		{name: "not filled", filled: 0, want: 300},
		{name: "partially filled", filled: 100, want: 200},
		{name: "fully filled", filled: 300, want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			order := entity.Order{QuantityOrdered: 300, QuantityFilled: tc.filled}
			require.Equal(t, tc.want, order.RemainingQuantity())
		})
	}
}
