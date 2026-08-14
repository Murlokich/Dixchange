package orderbook

import "github.com/google/btree"

// Introduced to unify asks and bids structures
// to provide getBestPriceLevel for side agnostic matchOrder method
type bookSide interface {
	getBestPriceLevel() (PriceLevel, bool)

	Len() int
	Get(PriceLevel) (PriceLevel, bool)
	Delete(item PriceLevel) (PriceLevel, bool)
	ReplaceOrInsert(PriceLevel) (PriceLevel, bool)
}

type asks struct {
	*btree.BTreeG[PriceLevel]
}

func (a *asks) getBestPriceLevel() (PriceLevel, bool) {
	return a.Min()
}

type bids struct {
	*btree.BTreeG[PriceLevel]
}

func (b *bids) getBestPriceLevel() (PriceLevel, bool) {
	return b.Max()
}
