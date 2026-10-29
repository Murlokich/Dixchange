package service

import (
	"github.com/Murlokich/Dixchange/internal/domain/entity"
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
	scaled, err := scaleToInt(amount, symbol.Precision)
	if err != nil {
		return 0, errors.Wrapf(err, "symbol %s", symbol.Code)
	}
	return scaled, nil
}

// Converts price to int64 using the price precision defined per pair.
// Returns an error instead of rounding if price has more decimal places than the pair supports.
func (c *Converter) ConvertPriceToInt(pair entity.Pair, price decimal.Decimal) (int64, error) {
	scaled, err := scaleToInt(price, pair.PricePrecision)
	if err != nil {
		return 0, errors.Wrapf(err, "pair %s/%s price", pair.Base.Code, pair.Quote.Code)
	}
	return scaled, nil
}

func scaleToInt(value decimal.Decimal, precision int) (int64, error) {
	//nolint:gosec // precision is smallint in postgres == no overflow
	shifted := value.Shift(int32(precision))
	if !shifted.IsInteger() {
		return 0, errors.Errorf("value %s has more decimal places than precision %d supports", value, precision)
	}

	scaled := shifted.BigInt()
	if !scaled.IsInt64() {
		return 0, errors.Errorf("value %s exceeds int64 range at precision %d", value, precision)
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
	priceInt, err := c.ConvertPriceToInt(order.MarketPair, price)
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "converter ConvertPriceToInt")
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
