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

// OrderCancelInfo is the part of an order needed to decide whether a cancel request
// may go to the matching engine.
type OrderCancelInfo struct {
	UserID uuid.UUID
	Status string
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

// GetDebitOrderedAmount returns the amount of the debit asset the whole order costs,
// which is what gets reserved when the order is created.
func (o *Order) GetDebitOrderedAmount() (int64, error) {
	return o.debitAmount(o.QuantityOrdered)
}

// GetDebitRemainingAmount returns the amount of the debit asset the unfilled part of the
// order costs, which is what is still reserved for it. Fills consume the reservation at
// the order price: a buyer's price improvement is released by the trade itself.
func (o *Order) GetDebitRemainingAmount() (int64, error) {
	return o.debitAmount(o.RemainingQuantity())
}

// IsActive reports whether the order can still trade.
func (o *Order) IsActive() bool {
	return IsActiveOrderStatus(o.Status)
}

// IsActiveOrderStatus reports whether an order in the status can still trade.
func IsActiveOrderStatus(status string) bool {
	return status == StatusOpen || status == StatusPartiallyFilled
}

// debitAmount returns the amount of the debit asset quantity (in base minor units) costs.
func (o *Order) debitAmount(quantity int64) (int64, error) {
	if o.Side == BuySide {
		amount, err := o.MarketPair.Notional(quantity, o.PricePerUnit)
		if err != nil {
			return 0, errors.Wrap(err, "market pair notional")
		}
		return amount, nil
	}
	return quantity, nil
}
