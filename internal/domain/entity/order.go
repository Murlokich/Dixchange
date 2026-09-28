package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	BuySide  = "buy"
	SellSide = "sell"

	StatusOpen            = "open"
	StatusPartiallyFilled = "partially_filled"
	StatusFilled          = "filled"
	StatusCancelled       = "cancelled"
)

type Order struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	MarketPair      Pair
	Side            string
	QuantityOrdered int64
	QuantityFilled  int64
	PricePerUnit    int64
	Status          string
	CreatedAt       time.Time
	LastUpdatedAt   time.Time
}

// RemainingQuantity returns the quantity still to be filled, in base minor units.
func (o *Order) RemainingQuantity() int64 {
	return o.QuantityOrdered - o.QuantityFilled
}

// GetDebitAsset returns symbol for asset that is deducted from the wallet.
func (o *Order) GetDebitAsset() (Symbol, error) {
	switch o.Side {
	case BuySide:
		return o.MarketPair.Quote, nil
	case SellSide:
		return o.MarketPair.Base, nil
	}
	return Symbol{}, errors.Errorf("invalid side value: %s", o.Side)
}

// GetCreditAsset returns symbol for asset that is added to the wallet.
func (o *Order) GetCreditAsset() (Symbol, error) {
	switch o.Side {
	case BuySide:
		return o.MarketPair.Base, nil
	case SellSide:
		return o.MarketPair.Quote, nil
	}
	return Symbol{}, errors.Errorf("invalid side value: %s", o.Side)
}

func (o *Order) GetDebitAmount() (int64, error) {
	if o.Side == BuySide {
		amount, err := o.MarketPair.Notional(o.QuantityOrdered, o.PricePerUnit)
		if err != nil {
			return 0, errors.Wrap(err, "market pair notional")
		}
		return amount, nil
	}
	return o.QuantityOrdered, nil
}
