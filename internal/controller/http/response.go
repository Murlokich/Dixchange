package http

import (
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/shopspring/decimal"
)

type LoginResponse struct {
	AccessToken string
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
