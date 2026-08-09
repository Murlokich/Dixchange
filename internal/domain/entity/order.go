package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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
	MarketPair      Pair
	Action          string
	QuantityOrdered decimal.Decimal
	QuantityFilled  decimal.Decimal
	PricePerUnit    decimal.Decimal
	Status          string
	CreatedAt       time.Time
	LastUpdatedAt   time.Time
}

// GetDebitAsset returns symbol for asset that is deducted from the wallet.
func (o *Order) GetDebitAsset() (Symbol, error) {
	switch o.Action {
	case BuyAction:
		return o.MarketPair.Quote, nil
	case SellAction:
		return o.MarketPair.Base, nil
	}
	return Symbol{}, errors.Errorf("invalid action value: %s", o.Action)
}

// GetCreditAsset returns symbol for asset that is added to the wallet.
func (o *Order) GetCreditAsset() (Symbol, error) {
	switch o.Action {
	case BuyAction:
		return o.MarketPair.Base, nil
	case SellAction:
		return o.MarketPair.Quote, nil
	}
	return Symbol{}, errors.Errorf("invalid action value: %s", o.Action)
}

func (o *Order) GetDebitAmount() decimal.Decimal {
	if o.Action == BuyAction {
		return o.PricePerUnit.Mul(o.QuantityOrdered)
	}
	return o.QuantityOrdered
}
