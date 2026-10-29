package http //nolint:testpackage // tests the unexported response mapping

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	gettrades "github.com/Murlokich/Dixchange/internal/domain/usecase/get_trades"
	"github.com/google/uuid"
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

func TestNewOrdersResponse_JSON(t *testing.T) {
	t.Parallel()

	btcUsd := entity.Pair{
		ID:             2,
		Base:           entity.Symbol{ID: 3, Code: "BTC", Precision: 8},
		Quote:          entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		LotSize:        1000,
		PriceTick:      5,
		PricePrecision: 2,
	}
	// a non-UTC zone checks that timestamps are converted to UTC
	cet := time.FixedZone("CET", 60*60)

	orders := []entity.Order{
		{
			ID:              uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e33"),
			UserID:          uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e44"),
			MarketPair:      btcUsd,
			Side:            entity.BuySide,
			QuantityOrdered: 123000,
			QuantityFilled:  23000,
			PricePerUnit:    6500050,
			Status:          entity.StatusOpen,
			CreatedAt:       time.Date(2026, 9, 26, 13, 4, 5, 0, cet),
			LastUpdatedAt:   time.Date(2026, 9, 26, 13, 30, 0, 0, cet),
		},
		{
			ID:              uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e55"),
			MarketPair:      btcUsd,
			Side:            entity.SellSide,
			QuantityOrdered: 1000,
			PricePerUnit:    10,
			Status:          entity.StatusOpen,
			CreatedAt:       time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		},
	}

	body, err := json.Marshal(newOrdersResponse(orders))

	require.NoError(t, err)
	require.JSONEq(t, `{
		"orders": [
			{
				"id": "019f7f2c-0caa-70ab-b1ec-414f744c3e33",
				"pair_id": 2,
				"side": "buy",
				"quantity_ordered": "0.00123000",
				"quantity_filled": "0.00023000",
				"remaining_quantity": "0.00100000",
				"price_per_unit": "65000.50",
				"status": "open",
				"created_at": "2026-09-26T12:04:05Z",
				"last_updated_at": "2026-09-26T12:30:00Z"
			},
			{
				"id": "019f7f2c-0caa-70ab-b1ec-414f744c3e55",
				"pair_id": 2,
				"side": "sell",
				"quantity_ordered": "0.00001000",
				"quantity_filled": "0.00000000",
				"remaining_quantity": "0.00001000",
				"price_per_unit": "0.10",
				"status": "open",
				"created_at": "2026-09-26T12:00:00Z",
				"last_updated_at": null
			}
		]
	}`, string(body))
}

func TestNewOrdersResponse_NoOrdersIsEmptyArray(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(newOrdersResponse(nil))

	require.NoError(t, err)
	require.JSONEq(t, `{"orders": []}`, string(body))
}

func TestNewTradesResponse_JSON(t *testing.T) {
	t.Parallel()

	btcUsd := entity.Pair{
		ID:             2,
		Base:           entity.Symbol{ID: 3, Code: "BTC", Precision: 8},
		Quote:          entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		LotSize:        1000,
		PriceTick:      5,
		PricePrecision: 2,
	}
	cet := time.FixedZone("CET", 60*60)
	cursor := uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3eb1")

	output := gettrades.Output{
		Trades: []gettrades.UserTrade{
			{
				Trade: entity.Trade{
					ID:         uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3eb1"),
					MarketPair: btcUsd,
					Price:      6500050,
					Quantity:   123000,
					CreatedAt:  time.Date(2026, 9, 28, 13, 4, 5, 0, cet),
				},
				Party: entity.UserTradeParty{
					Side:    entity.SellSide,
					OrderID: uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e0a"),
				},
				Total: 7995061,
			},
		},
		NextCursor: &cursor,
	}

	body, err := json.Marshal(newTradesResponse(output))

	require.NoError(t, err)
	require.JSONEq(t, `{
		"trades": [
			{
				"id": "019f7f2c-0caa-70ab-b1ec-414f744c3eb1",
				"pair_id": 2,
				"order_id": "019f7f2c-0caa-70ab-b1ec-414f744c3e0a",
				"side": "sell",
				"amount": "0.00123000",
				"price": "65000.50",
				"total": "79950.61",
				"executed_at": "2026-09-28T12:04:05Z"
			}
		],
		"next_cursor": "019f7f2c-0caa-70ab-b1ec-414f744c3eb1"
	}`, string(body))
}

func TestNewTradesResponse_LastPageHasNullCursor(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(newTradesResponse(gettrades.Output{}))

	require.NoError(t, err)
	require.JSONEq(t, `{"trades": [], "next_cursor": null}`, string(body))
}
