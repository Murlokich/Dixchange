package service

import (
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidNumberOfLots       = errors.New("Quantity provided doesn't match to integer number of lot sizes")
	ErrInvalidNumberOfPriceTicks = errors.New("Price provided doesn't match to integer number of price ticks")
)

type Converter struct {
}

// Converts decimal to int64 using the precision defined per symbol.
// Returns an error instead of rounding if amount has more decimal places than the symbol supports.
func (c *Converter) ConvertAmountToInt(symbol entity.Symbol, amount decimal.Decimal) (int64, error) {
	//nolint:gosec // symbol precision is smallint in postgres == no overflow
	shifted := amount.Shift(int32(symbol.Precision))
	if !shifted.IsInteger() {
		return 0, errors.Errorf(
			"amount %s has more decimal places than symbol %s supports (precision %d)",
			amount,
			symbol.Code,
			symbol.Precision,
		)
	}

	scaled := shifted.BigInt()
	if !scaled.IsInt64() {
		return 0, errors.Errorf(
			"amount %s exceeds int64 range for symbol %s (precision %d)",
			amount,
			symbol.Code,
			symbol.Precision,
		)
	}
	return scaled.Int64(), nil
}

func (c *Converter) DeriveOrderWithQuantitativeFields(
	order entity.Order,
	quantity decimal.Decimal,
	price decimal.Decimal,
) (entity.Order, error) {
	quantityInt, err := c.ConvertAmountToInt(order.MarketPair.Base, quantity)
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "converter ConvertAmountToInt quantity")
	}
	if quantityInt%order.MarketPair.LotSize != 0 {
		return entity.Order{}, ErrInvalidNumberOfLots
	}
	priceInt, err := c.ConvertAmountToInt(order.MarketPair.Quote, price)
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "converter ConvertAmountToInt price")
	}
	if priceInt%order.MarketPair.PriceTick != 0 {
		return entity.Order{}, ErrInvalidNumberOfPriceTicks
	}
	order.QuantityOrdered = quantityInt
	order.PricePerUnit = priceInt
	return order, nil
}

func NewConverter() *Converter {
	return &Converter{}
}
