//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

// An incoming order trades at the resting order's price. A bid reserves quantity * its own
// price, so a trade at a better (lower) price gets back the difference; an ask reserves
// only the base it sells, so it has nothing to get back and is paid the better price.
// Each user also has a resting order outside the trade: its reservation must stay, so a
// refund of everything reserved would show.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestFundsRefund_Bid_BetterAsk_RefundsExcess(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	h.mustPlaceOrder(t, buyer, entity.BuySide, "2", "5")   // buy 2 USD at 5 KCN: rests, reserves 10 KCN
	h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN: rests
	h.mustPlaceOrder(t, buyer, entity.BuySide, "3", "8")   // buy 3 USD at 8 KCN: reserves 24 KCN, trades 3 at 7

	// The buyer paid 3 * 7 = 21 KCN of the 24 reserved and got 3 back; only the resting
	// bid's 10 KCN stay reserved.
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 69, Reserved: 10}, buyerBalances.Quote)
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 21}, sellerBalances.Quote)
}

func TestFundsRefund_Ask_BetterBid_NothingToRefund(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	h.mustPlaceOrder(t, seller, entity.SellSide, "2", "9") // sell 2 USD at 9 KCN: rests, reserves 2 USD
	h.mustPlaceOrder(t, buyer, entity.BuySide, "3", "8")   // buy 3 USD at 8 KCN: rests, reserves 24 KCN
	h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN: reserves 3 USD, trades 3 at 8

	// The seller gave the 3 USD reserved and got 3 * 8 = 24 KCN, not 21; only the resting
	// ask's 2 USD stay reserved.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 500, Reserved: 200}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 24}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 76}, buyerBalances.Quote)
}
