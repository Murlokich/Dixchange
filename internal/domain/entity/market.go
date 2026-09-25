package entity

import (
	"math/big"

	"github.com/pkg/errors"
)

// decimalBase is the base of the scales: a value at scale s is its integer times 10^-s.
const decimalBase = 10

type Pair struct {
	ID    int
	Base  Symbol
	Quote Symbol
	// LotSize is the quantity step, in base minor units.
	LotSize int64
	// PriceTick is the price step, at PricePrecision scale.
	PriceTick int64
	// PricePrecision is the scale of every price on the pair. A price is quote per one
	// whole base, not an amount of either asset, so it is owned by the pair.
	PricePrecision int
}

// Notional returns the cost of quantity (in base minor units) at price (at the pair
// price precision) in quote minor units. See numeric_types.md for the derivation.
//
// The pair parameters guarantee the result is exact for any whole number of lots at a
// whole number of ticks, so a remainder means the inputs or the pair are invalid.
func (p Pair) Notional(quantity, price int64) (int64, error) {
	// quantity * price has scale b + p; the result must have scale q.
	k := p.Base.Precision + p.PricePrecision - p.Quote.Precision

	notional := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(price))
	if k >= 0 {
		scale := new(big.Int).Exp(big.NewInt(decimalBase), big.NewInt(int64(k)), nil)
		remainder := new(big.Int)
		notional.QuoRem(notional, scale, remainder)
		if remainder.Sign() != 0 {
			return 0, errors.Errorf(
				"notional of quantity %d at price %d is not a whole number of %s minor units",
				quantity, price, p.Quote.Code,
			)
		}
	} else {
		scale := new(big.Int).Exp(big.NewInt(decimalBase), big.NewInt(int64(-k)), nil)
		notional.Mul(notional, scale)
	}

	if !notional.IsInt64() {
		return 0, errors.Errorf("notional of quantity %d at price %d overflows int64", quantity, price)
	}
	return notional.Int64(), nil
}

type Symbol struct {
	ID        int
	Code      string
	Precision int
}
