package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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
	QuantityOrdered int64
	QuantityFilled  int64
	PricePerUnit    int64
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

func (o *Order) GetDebitAmount() (int64, error) {
	if o.Action == BuyAction {
		amount, err := o.MarketPair.Notional(o.QuantityOrdered, o.PricePerUnit)
		if err != nil {
			return 0, errors.Wrap(err, "market pair notional")
		}
		return amount, nil
	}
	return o.QuantityOrdered, nil
}
