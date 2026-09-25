package entity_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

// Pairs below are all valid: (LotSize * PriceTick) % 10^k == 0, see numeric_types.md.
//
//nolint:gochecknoglobals // acceptable in test
var (
	// Base with minor units, quote without: k = 2 + 0 - 0 = 2.
	usdKcn = entity.Pair{
		Base:      entity.Symbol{Code: "USD", Precision: 2},
		Quote:     entity.Symbol{Code: "KCN", Precision: 0},
		LotSize:   100, // 1 USD
		PriceTick: 1,   // 1 KCN
	}
	// Inverse of usdKcn: k = 0 + 2 - 2 = 0, scales cancel out.
	kcnUsd = entity.Pair{
		Base:           entity.Symbol{Code: "KCN", Precision: 0},
		Quote:          entity.Symbol{Code: "USD", Precision: 2},
		LotSize:        1, // 1 KCN
		PriceTick:      1, // 0.01 USD
		PricePrecision: 2,
	}
	// Coarse price tick: k = 0 + 0 - 2 = -2, result is scaled up.
	kcnUsdWholeDollars = entity.Pair{
		Base:      entity.Symbol{Code: "KCN", Precision: 0},
		Quote:     entity.Symbol{Code: "USD", Precision: 2},
		LotSize:   1, // 1 KCN
		PriceTick: 1, // 1 USD
	}
	// Expensive base: k = 8 + 2 - 8 = 2.
	btcUsdt = entity.Pair{
		Base:           entity.Symbol{Code: "BTC", Precision: 8},
		Quote:          entity.Symbol{Code: "USDT", Precision: 8},
		LotSize:        1000, // 0.00001 BTC
		PriceTick:      1,    // 0.01 USDT
		PricePrecision: 2,
	}
	// Close-valued fiat pair with a fine tick: k = 2 + 5 - 2 = 5.
	eurUsd = entity.Pair{
		Base:           entity.Symbol{Code: "EUR", Precision: 2},
		Quote:          entity.Symbol{Code: "USD", Precision: 2},
		LotSize:        100000, // 1000 EUR
		PriceTick:      1,      // 0.00001 USD
		PricePrecision: 5,
	}
	// Close-valued crypto pair: k = 8 + 5 - 8 = 5.
	ethBtc = entity.Pair{
		Base:           entity.Symbol{Code: "ETH", Precision: 8},
		Quote:          entity.Symbol{Code: "BTC", Precision: 8},
		LotSize:        100000, // 0.001 ETH
		PriceTick:      1,      // 0.00001 BTC
		PricePrecision: 5,
	}
	// Cheap base: k = 8 + 8 - 8 = 8.
	shibUsdt = entity.Pair{
		Base:           entity.Symbol{Code: "SHIB", Precision: 8},
		Quote:          entity.Symbol{Code: "USDT", Precision: 8},
		LotSize:        100000000, // 1 SHIB
		PriceTick:      1,         // 0.00000001 USDT
		PricePrecision: 8,
	}
)

func TestPair_Notional(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pair     entity.Pair
		quantity int64
		price    int64
		want     int64
	}{
		{
			name:     "USD/KCN: 1 USD at 1 KCN costs 1 KCN",
			pair:     usdKcn,
			quantity: 100,
			price:    1,
			want:     1,
		},
		{
			name:     "USD/KCN: 3 USD at 5 KCN costs 15 KCN",
			pair:     usdKcn,
			quantity: 300,
			price:    5,
			want:     15,
		},
		{
			name:     "KCN/USD: 5 KCN at 0.37 USD costs 1.85 USD",
			pair:     kcnUsd,
			quantity: 5,
			price:    37,
			want:     185,
		},
		{
			name:     "KCN/USD whole dollars: 5 KCN at 3 USD costs 15.00 USD",
			pair:     kcnUsdWholeDollars,
			quantity: 5,
			price:    3,
			want:     1500,
		},
		{
			name:     "BTC/USDT: 0.00123 BTC at 65000.50 costs 79.95061500 USDT",
			pair:     btcUsdt,
			quantity: 123000,
			price:    6500050,
			want:     7995061500,
		},
		{
			name:     "EUR/USD: 1000 EUR at 1.08345 costs 1083.45 USD",
			pair:     eurUsd,
			quantity: 100000,
			price:    108345,
			want:     108345,
		},
		{
			name:     "ETH/BTC: 0.5 ETH at 0.05123 BTC costs 0.025615 BTC",
			pair:     ethBtc,
			quantity: 50000000,
			price:    5123,
			want:     2561500,
		},
		{
			name: "SHIB/USDT: 100M SHIB at 0.00001234 costs 1234 USDT, " +
				"intermediate product exceeds int64",
			pair:     shibUsdt,
			quantity: 10000000000000000,
			price:    1234,
			want:     123400000000,
		},
		{
			name:     "zero quantity costs nothing",
			pair:     btcUsdt,
			quantity: 0,
			price:    6500050,
			want:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.pair.Notional(tc.quantity, tc.price)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPair_Notional_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		pair        entity.Pair
		quantity    int64
		price       int64
		errContains string
	}{
		{
			name:        "quantity not a whole number of lots: 1 cent at 1 KCN",
			pair:        usdKcn,
			quantity:    1,
			price:       1,
			errContains: "not a whole number of KCN minor units",
		},
		{
			name:        "result overflows int64 when scaling down",
			pair:        btcUsdt,
			quantity:    1000000000000000, // 10M BTC
			price:       1000000000,       // 10M USDT
			errContains: "overflows int64",
		},
		{
			name:        "result overflows int64 when scaling up",
			pair:        kcnUsdWholeDollars,
			quantity:    100000000000000000,
			price:       1000,
			errContains: "overflows int64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.pair.Notional(tc.quantity, tc.price)
			require.ErrorContains(t, err, tc.errContains)
			require.Zero(t, got)
		})
	}
}
