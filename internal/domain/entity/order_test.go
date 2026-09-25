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
		action    string
		wantAsset entity.Symbol
		wantErr   bool
	}{
		{name: "buy", action: entity.BuyAction, wantAsset: testPair.Quote},
		{name: "sell", action: entity.SellAction, wantAsset: testPair.Base},
		{name: "invalid action", action: "gibberish", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order := entity.Order{MarketPair: testPair, Action: tc.action}
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

func TestOrder_GetDebitAmount(t *testing.T) {
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
				Action:          entity.BuyAction,
				QuantityOrdered: 123000,
				PricePerUnit:    6500050,
			},
			want: 7995061500,
		},
		{
			name: "buy debits notional in quote minor units: 1 USD at 1 KCN",
			order: entity.Order{
				MarketPair:      usdKcn,
				Action:          entity.BuyAction,
				QuantityOrdered: 100,
				PricePerUnit:    1,
			},
			want: 1,
		},
		{
			name: "sell debits quantity in base minor units",
			order: entity.Order{
				MarketPair:      btcUsdt,
				Action:          entity.SellAction,
				QuantityOrdered: 123000,
				PricePerUnit:    6500050,
			},
			want: 123000,
		},
		{
			name: "buy propagates pair notional error",
			order: entity.Order{
				MarketPair:      usdKcn,
				Action:          entity.BuyAction,
				QuantityOrdered: 1,
				PricePerUnit:    1,
			},
			errContains: "market pair notional",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			amount, err := tc.order.GetDebitAmount()
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

func TestOrder_GetCreditAsset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		action    string
		wantAsset entity.Symbol
		wantErr   bool
	}{
		{name: "buy", action: entity.BuyAction, wantAsset: testPair.Base},
		{name: "sell", action: entity.SellAction, wantAsset: testPair.Quote},
		{name: "invalid action", action: "gibberish", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order := entity.Order{MarketPair: testPair, Action: tc.action}
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
