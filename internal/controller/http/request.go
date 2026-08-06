package http

import (
	"github.com/shopspring/decimal"
)

type LoginRequest struct {
	Password string `json:"password" binding:"required,min=8,max=256"`
	Email    string `json:"email" binding:"required,min=6,max=254"`
}

type RegisterRequest struct {
	Password string `json:"password" binding:"required,min=8,max=256"`
	Email    string `json:"email" binding:"required,min=6,max=254"`
}

type DepositRequest struct {
	SymbolID int             `json:"symbol_id" binding:"required,min=0"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
}

type CreateOrderRequest struct {
	MarketPairID    int             `json:"market_pair_id" binding:"required,min=0"`
	Action          string          `json:"action" binding:"required,oneof=buy sell"`
	QuantityOrdered decimal.Decimal `json:"quantity_ordered"`
	PricePerUnit    decimal.Decimal `json:"price_per_unit"`
}
