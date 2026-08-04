package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	BuyAction  = "buy"
	SellAction = "sell"

	StatusOpen      = "open"
	StatusClosed    = "closed"
	StatusCancelled = "cancelled"
)

type Order struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	MarketPairID    int
	Action          string
	QuantityOrdered decimal.Decimal
	QuantityOpen    decimal.Decimal
	PricePerUnit    decimal.Decimal
	Status          string
	CreatedAt       time.Time
	LastUpdatedAt   time.Time
}
