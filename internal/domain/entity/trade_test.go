package entity_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTrade_Notional(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		trade       entity.Trade
		want        int64
		errContains string
	}{
		{
			name:  "BTC/USDT: 0.00123 BTC at 65000.50 costs 79.95061500 USDT",
			trade: entity.Trade{MarketPair: btcUsdt, Quantity: 123000, Price: 6500050},
			want:  7995061500,
		},
		{
			name:  "USD/KCN: 3 USD at 5 KCN costs 15 KCN",
			trade: entity.Trade{MarketPair: usdKcn, Quantity: 300, Price: 5},
			want:  15,
		},
		{
			name:        "pair notional error is propagated",
			trade:       entity.Trade{MarketPair: usdKcn, Quantity: 1, Price: 1},
			errContains: "market pair notional",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.trade.Notional()
			if tc.errContains != "" {
				require.ErrorContains(t, err, tc.errContains)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTrade_GetExcessReservedAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		trade       entity.Trade
		want        int64
		errContains string
	}{
		{
			name: "BTC/USDT: bid at 65000.50 filled at 65000.00 frees 0.50 per BTC",
			trade: entity.Trade{
				MarketPair: btcUsdt,
				Bid:        entity.TradeParty{OrderPrice: 6500050},
				Price:      6500000,
				Quantity:   123000,
			},
			want: 61500, // 0.00123 BTC * 0.50 = 0.00061500 USDT
		},
		{
			name: "USD/KCN: bid at 7 KCN filled at 5 KCN frees 2 KCN per USD",
			trade: entity.Trade{
				MarketPair: usdKcn,
				Bid:        entity.TradeParty{OrderPrice: 7},
				Price:      5,
				Quantity:   300,
			},
			want: 6,
		},
		{
			name: "bid filled at its own price frees nothing",
			trade: entity.Trade{
				MarketPair: btcUsdt,
				Bid:        entity.TradeParty{OrderPrice: 6500050},
				Price:      6500050,
				Quantity:   123000,
			},
			want: 0,
		},
		{
			name: "trade price above the bid price frees nothing",
			trade: entity.Trade{
				MarketPair: btcUsdt,
				Bid:        entity.TradeParty{OrderPrice: 6500000},
				Price:      6500050,
				Quantity:   123000,
			},
			want: 0,
		},
		{
			name: "pair notional error is propagated",
			trade: entity.Trade{
				MarketPair: usdKcn,
				Bid:        entity.TradeParty{OrderPrice: 7},
				Price:      5,
				Quantity:   1,
			},
			errContains: "market pair notional",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.trade.GetExcessReservedAmount()
			if tc.errContains != "" {
				require.ErrorContains(t, err, tc.errContains)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTrade_PartyOf(t *testing.T) {
	t.Parallel()

	buyer, seller, stranger := uuid.New(), uuid.New(), uuid.New()
	bidOrder, askOrder := uuid.New(), uuid.New()
	trade := entity.Trade{
		Bid: entity.TradeParty{UserID: buyer, OrderID: bidOrder},
		Ask: entity.TradeParty{UserID: seller, OrderID: askOrder},
	}
	selfTrade := entity.Trade{
		Bid: entity.TradeParty{UserID: buyer, OrderID: bidOrder},
		Ask: entity.TradeParty{UserID: buyer, OrderID: askOrder},
	}

	testCases := []struct {
		name    string
		trade   entity.Trade
		userID  uuid.UUID
		want    entity.UserTradeParty
		wantErr bool
	}{
		{name: "buyer", trade: trade, userID: buyer, want: entity.UserTradeParty{Side: entity.BuySide, OrderID: bidOrder}},
		{
			name:   "seller",
			trade:  trade,
			userID: seller,
			want:   entity.UserTradeParty{Side: entity.SellSide, OrderID: askOrder},
		},
		{
			name:   "self trade is reported as buy",
			trade:  selfTrade,
			userID: buyer,
			want:   entity.UserTradeParty{Side: entity.BuySide, OrderID: bidOrder},
		},
		{name: "not a participant", trade: trade, userID: stranger, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.trade.PartyByUserID(tc.userID)
			if tc.wantErr {
				require.ErrorIs(t, err, entity.ErrNotTradeParticipant)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
