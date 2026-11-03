package orderbook

import (
	"cmp"
	"container/list"
)

type PriceLevel struct {
	Price  int64
	Orders *list.List
}

func less(x, y PriceLevel) bool {
	return cmp.Less(x.Price, y.Price)
}

// Introduced to unify asks and bids structures
// to provide getBestPriceLevel for side agnostic matchOrder method
type bookSide interface {
	getBestPriceLevel() (PriceLevel, bool)

	Len() int
	Get(PriceLevel) (PriceLevel, bool)
	Delete(item PriceLevel) (PriceLevel, bool)
	ReplaceOrInsert(PriceLevel) (PriceLevel, bool)
}
