package http

import (
	"time"

	"github.com/dixchange/Monolith/internal/domain/entity"
	gettrades "github.com/dixchange/Monolith/internal/domain/usecase/get_trades"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type LoginResponse struct {
	AccessToken string
}

type PairBalanceResponse struct {
	PairID int             `json:"pair_id"`
	Base   BalanceResponse `json:"base"`
	Quote  BalanceResponse `json:"quote"`
}

type BalanceResponse struct {
	// Main is the available amount as a decimal at the symbol precision.
	Main string `json:"main"`
	// Reserved is the amount locked by open orders as a decimal at the symbol precision.
	Reserved string `json:"reserved"`
}

type OrdersResponse struct {
	Orders []OrderResponse `json:"orders"`
}

type OrderResponse struct {
	ID     uuid.UUID `json:"id"`
	PairID int       `json:"pair_id"`
	Side   string    `json:"side"`
	// Quantities are decimals at the base symbol precision.
	QuantityOrdered   string `json:"quantity_ordered"`
	QuantityFilled    string `json:"quantity_filled"`
	RemainingQuantity string `json:"remaining_quantity"`
	// PricePerUnit is a decimal at the pair price precision.
	PricePerUnit string `json:"price_per_unit"`
	Status       string `json:"status"`
	// Timestamps are RFC 3339 in UTC. LastUpdatedAt is null until the order is first updated.
	CreatedAt     string  `json:"created_at"`
	LastUpdatedAt *string `json:"last_updated_at"`
}

type TradesResponse struct {
	Trades []TradeResponse `json:"trades"`
	// NextCursor is passed as ?cursor= to load the next page; null when there are no more trades.
	NextCursor *uuid.UUID `json:"next_cursor"`
}

// TradeResponse is a trade as seen by the requesting user: their side and their order.
type TradeResponse struct {
	ID      uuid.UUID `json:"id"`
	PairID  int       `json:"pair_id"`
	OrderID uuid.UUID `json:"order_id"`
	Side    string    `json:"side"`
	// Amount is a decimal at the base symbol precision.
	Amount string `json:"amount"`
	// Price is the execution price as a decimal at the pair price precision.
	Price string `json:"price"`
	// Total is what the buyer paid as a decimal at the quote symbol precision.
	Total string `json:"total"`
	// ExecutedAt is RFC 3339 in UTC.
	ExecutedAt string `json:"executed_at"`
}

type MarketDataResponse struct {
	Pairs []PairResponse `json:"pairs"`
}

type PairResponse struct {
	ID int `json:"id"`
	// Symbols are ordered base first, then quote.
	Symbols []SymbolResponse `json:"symbols"`
	// LotSize is the quantity step as a decimal in the base symbol.
	LotSize string `json:"lot_size"`
	// PriceTick is the price step as a decimal at the pair price precision.
	PriceTick      string `json:"price_tick"`
	PricePrecision int    `json:"price_precision"`
}

type SymbolResponse struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Precision int    `json:"precision"`
}

func newMarketDataResponse(pairs []entity.Pair) MarketDataResponse {
	resp := MarketDataResponse{Pairs: make([]PairResponse, 0, len(pairs))}
	for _, p := range pairs {
		//nolint:gosec // precisions are smallint in postgres == no overflow
		basePrecision, pricePrecision := int32(p.Base.Precision), int32(p.PricePrecision)
		resp.Pairs = append(resp.Pairs, PairResponse{
			ID: p.ID,
			Symbols: []SymbolResponse{
				{ID: p.Base.ID, Code: p.Base.Code, Precision: p.Base.Precision},
				{ID: p.Quote.ID, Code: p.Quote.Code, Precision: p.Quote.Precision},
			},
			LotSize:        decimal.New(p.LotSize, -basePrecision).StringFixed(basePrecision),
			PriceTick:      decimal.New(p.PriceTick, -pricePrecision).StringFixed(pricePrecision),
			PricePrecision: p.PricePrecision,
		})
	}
	return resp
}

func newPairBalanceResponse(pair entity.Pair, base, quote entity.Balance) PairBalanceResponse {
	//nolint:gosec // precisions are smallint in postgres == no overflow
	basePrecision, quotePrecision := int32(pair.Base.Precision), int32(pair.Quote.Precision)
	return PairBalanceResponse{
		PairID: pair.ID,
		Base: BalanceResponse{
			Main:     decimal.New(base.Main, -basePrecision).StringFixed(basePrecision),
			Reserved: decimal.New(base.Reserved, -basePrecision).StringFixed(basePrecision),
		},
		Quote: BalanceResponse{
			Main:     decimal.New(quote.Main, -quotePrecision).StringFixed(quotePrecision),
			Reserved: decimal.New(quote.Reserved, -quotePrecision).StringFixed(quotePrecision),
		},
	}
}

func newOrdersResponse(orders []entity.Order) OrdersResponse {
	resp := OrdersResponse{Orders: make([]OrderResponse, 0, len(orders))}
	for _, o := range orders {
		//nolint:gosec // precisions are smallint in postgres == no overflow
		basePrecision, pricePrecision := int32(o.MarketPair.Base.Precision), int32(o.MarketPair.PricePrecision)

		var lastUpdatedAt *string
		if !o.LastUpdatedAt.IsZero() {
			formatted := o.LastUpdatedAt.UTC().Format(time.RFC3339)
			lastUpdatedAt = &formatted
		}

		resp.Orders = append(resp.Orders, OrderResponse{
			ID:                o.ID,
			PairID:            o.MarketPair.ID,
			Side:              o.Side,
			QuantityOrdered:   decimal.New(o.QuantityOrdered, -basePrecision).StringFixed(basePrecision),
			QuantityFilled:    decimal.New(o.QuantityFilled, -basePrecision).StringFixed(basePrecision),
			RemainingQuantity: decimal.New(o.RemainingQuantity(), -basePrecision).StringFixed(basePrecision),
			PricePerUnit:      decimal.New(o.PricePerUnit, -pricePrecision).StringFixed(pricePrecision),
			Status:            o.Status,
			CreatedAt:         o.CreatedAt.UTC().Format(time.RFC3339),
			LastUpdatedAt:     lastUpdatedAt,
		})
	}
	return resp
}

func newTradesResponse(output gettrades.Output) TradesResponse {
	resp := TradesResponse{Trades: make([]TradeResponse, 0, len(output.Trades))}
	for _, ut := range output.Trades {
		pair := ut.Trade.MarketPair
		//nolint:gosec // precisions are smallint in postgres == no overflow
		var (
			basePrecision  = int32(pair.Base.Precision)
			quotePrecision = int32(pair.Quote.Precision)
			pricePrecision = int32(pair.PricePrecision)
		)

		resp.Trades = append(resp.Trades, TradeResponse{
			ID:         ut.Trade.ID,
			PairID:     pair.ID,
			OrderID:    ut.Party.OrderID,
			Side:       ut.Party.Side,
			Amount:     decimal.New(ut.Trade.Quantity, -basePrecision).StringFixed(basePrecision),
			Price:      decimal.New(ut.Trade.Price, -pricePrecision).StringFixed(pricePrecision),
			Total:      decimal.New(ut.Total, -quotePrecision).StringFixed(quotePrecision),
			ExecutedAt: ut.Trade.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	resp.NextCursor = output.NextCursor
	return resp
}
