package http //nolint:testpackage // tests the unexported response mapping

import (
	"encoding/json"
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

func TestNewMarketDataResponse_JSON(t *testing.T) {
	t.Parallel()

	pairs := []entity.Pair{
		{
			ID:        1,
			Base:      entity.Symbol{ID: 1, Code: "USD", Precision: 2},
			Quote:     entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
			LotSize:   100,
			PriceTick: 1,
		},
		{
			ID:             2,
			Base:           entity.Symbol{ID: 3, Code: "BTC", Precision: 8},
			Quote:          entity.Symbol{ID: 1, Code: "USD", Precision: 2},
			LotSize:        1000,
			PriceTick:      5,
			PricePrecision: 2,
		},
	}

	body, err := json.Marshal(newMarketDataResponse(pairs))

	require.NoError(t, err)
	require.JSONEq(t, `{
		"pairs": [
			{
				"id": 1,
				"symbols": [{"id": 1, "code": "USD", "precision": 2}, {"id": 2, "code": "KCN", "precision": 0}],
				"lot_size": "1.00",
				"price_tick": "1",
				"price_precision": 0
			},
			{
				"id": 2,
				"symbols": [{"id": 3, "code": "BTC", "precision": 8}, {"id": 1, "code": "USD", "precision": 2}],
				"lot_size": "0.00001000",
				"price_tick": "0.05",
				"price_precision": 2
			}
		]
	}`, string(body))
}

func TestNewMarketDataResponse_NoPairsIsEmptyArray(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(newMarketDataResponse(nil))

	require.NoError(t, err)
	require.JSONEq(t, `{"pairs": []}`, string(body))
}

func TestNewPairBalanceResponse_JSON(t *testing.T) {
	t.Parallel()

	pair := entity.Pair{
		ID:    1,
		Base:  entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		Quote: entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
	}
	base := entity.Balance{SymbolID: 1, Main: 1050, Reserved: 0}
	quote := entity.Balance{SymbolID: 2, Main: 7, Reserved: 3}

	body, err := json.Marshal(newPairBalanceResponse(pair, base, quote))

	require.NoError(t, err)
	require.JSONEq(t, `{
		"pair_id": 1,
		"base": {"main": "10.50", "reserved": "0.00"},
		"quote": {"main": "7", "reserved": "3"}
	}`, string(body))
}
