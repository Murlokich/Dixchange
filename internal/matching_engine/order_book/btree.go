package orderbook

import "github.com/google/btree"

type asks struct {
	*btree.BTreeG[PriceLevel]
}

func newAsks(degree int) *asks {
	return &asks{btree.NewG[PriceLevel](degree, less)}
}

func (a *asks) getBestPriceLevel() (PriceLevel, bool) {
	return a.Min()
}

type bids struct {
	*btree.BTreeG[PriceLevel]
}

func newBids(degree int) *bids {
	return &bids{btree.NewG[PriceLevel](degree, less)}
}

func (b *bids) getBestPriceLevel() (PriceLevel, bool) {
	return b.Max()
}
