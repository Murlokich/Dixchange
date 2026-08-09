package entity_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/shopspring/decimal"
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

func TestOrder_GetDebitNumber(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		action string
		want   decimal.Decimal
	}{
		{
			name:   "buy debits price times quantity (quote asset)",
			action: entity.BuyAction,
			want:   decimal.NewFromInt(2000), // 100 * 20
		},
		{
			name:   "sell debits quantity (base asset)",
			action: entity.SellAction,
			want:   decimal.NewFromInt(20),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order := entity.Order{
				Action:          tc.action,
				PricePerUnit:    decimal.NewFromInt(100),
				QuantityOrdered: decimal.NewFromInt(20),
			}
			require.True(t, tc.want.Equal(order.GetDebitAmount()))
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
