//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
)

// An incoming order trades with the resting orders it crosses, best price first and, at
// one price, oldest first; each trade is at the resting order's price. Whatever the
// incoming order doesn't fill rests in the book. Each test checks the trades, the orders
// and both users' funds.
//
// A user only ever trades on one side: the engine doesn't prevent self-trades yet.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestMatching_NoCross_BothRest(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "3", "6")   // buy 3 USD at 6 KCN: below the ask

	assert.Empty(t, h.trades(t))
	assert.Equal(t, orderRow{ID: askID, Side: entity.SellSide, Price: 7, QuantityOrdered: 300,
		Status: entity.StatusOpen}, h.order(t, askID))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 6, QuantityOrdered: 300,
		Status: entity.StatusOpen}, h.order(t, bidID))

	// Both keep their reservations: 3 USD for the ask, 3 * 6 = 18 KCN for the bid.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700, Reserved: 300}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 82, Reserved: 18}, buyerBalances.Quote)
}

func TestMatching_EqualQuantities_BothFilled(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "3", "7")   // buy 3 USD at 7 KCN

	assert.Equal(t, []tradeRow{{BuyOrderID: bidID, SellOrderID: askID, Price: 7, Quantity: 300}}, h.trades(t))
	assert.Equal(t, orderRow{ID: askID, Side: entity.SellSide, Price: 7, QuantityOrdered: 300,
		QuantityFilled: 300, Status: entity.StatusFilled}, h.order(t, askID))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 7, QuantityOrdered: 300,
		QuantityFilled: 300, Status: entity.StatusFilled}, h.order(t, bidID))

	// 3 USD for 3 * 7 = 21 KCN; nothing stays reserved.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 21}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 79}, buyerBalances.Quote)
}

func TestMatching_IncomingBigger_RemainderRests(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "2", "7") // sell 2 USD at 7 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "5", "7")   // buy 5 USD at 7 KCN: 2 fill, 3 rest

	assert.Equal(t, []tradeRow{{BuyOrderID: bidID, SellOrderID: askID, Price: 7, Quantity: 200}}, h.trades(t))
	assert.Equal(t, orderRow{ID: askID, Side: entity.SellSide, Price: 7, QuantityOrdered: 200,
		QuantityFilled: 200, Status: entity.StatusFilled}, h.order(t, askID))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 7, QuantityOrdered: 500,
		QuantityFilled: 200, Status: entity.StatusPartiallyFilled}, h.order(t, bidID))

	// 2 USD for 2 * 7 = 14 KCN. The bid reserved 5 * 7 = 35 KCN; 3 * 7 = 21 stay reserved
	// for the 3 USD still resting.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 800}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 14}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 200}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 65, Reserved: 21}, buyerBalances.Quote)
}

func TestMatching_IncomingSmaller_RestingPartiallyFilled(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "5", "7") // sell 5 USD at 7 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "2", "7")   // buy 2 USD at 7 KCN: fills 2 of the 5

	assert.Equal(t, []tradeRow{{BuyOrderID: bidID, SellOrderID: askID, Price: 7, Quantity: 200}}, h.trades(t))
	assert.Equal(t, orderRow{ID: askID, Side: entity.SellSide, Price: 7, QuantityOrdered: 500,
		QuantityFilled: 200, Status: entity.StatusPartiallyFilled}, h.order(t, askID))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 7, QuantityOrdered: 200,
		QuantityFilled: 200, Status: entity.StatusFilled}, h.order(t, bidID))

	// 2 USD for 2 * 7 = 14 KCN; the 3 USD still resting stay reserved.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 500, Reserved: 300}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 14}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 200}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 86}, buyerBalances.Quote)
}

func TestMatching_SamePrice_OldestFirst(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	ask1 := h.mustPlaceOrder(t, seller, entity.SellSide, "1", "7") // sell 1 USD at 7 KCN
	ask2 := h.mustPlaceOrder(t, seller, entity.SellSide, "2", "7") // sell 2 USD at 7 KCN
	ask3 := h.mustPlaceOrder(t, seller, entity.SellSide, "2", "7") // sell 2 USD at 7 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "4", "7")  // buy 4 USD at 7 KCN: 1 + 2 + 1 of 2

	assert.Equal(t, []tradeRow{
		{BuyOrderID: bidID, SellOrderID: ask1, Price: 7, Quantity: 100},
		{BuyOrderID: bidID, SellOrderID: ask2, Price: 7, Quantity: 200},
		{BuyOrderID: bidID, SellOrderID: ask3, Price: 7, Quantity: 100},
	}, h.trades(t))
	assert.Equal(t, orderRow{ID: ask1, Side: entity.SellSide, Price: 7, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, ask1))
	assert.Equal(t, orderRow{ID: ask2, Side: entity.SellSide, Price: 7, QuantityOrdered: 200,
		QuantityFilled: 200, Status: entity.StatusFilled}, h.order(t, ask2))
	assert.Equal(t, orderRow{ID: ask3, Side: entity.SellSide, Price: 7, QuantityOrdered: 200,
		QuantityFilled: 100, Status: entity.StatusPartiallyFilled}, h.order(t, ask3))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 7, QuantityOrdered: 400,
		QuantityFilled: 400, Status: entity.StatusFilled}, h.order(t, bidID))

	// 4 USD for 4 * 7 = 28 KCN; the seller's last 1 USD still rests.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 500, Reserved: 100}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 28}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 400}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 72}, buyerBalances.Quote)
}

func TestMatching_Bid_SweepsAskLevels(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	// Placed out of price order, so only price decides which fills first.
	ask9 := h.mustPlaceOrder(t, seller, entity.SellSide, "1", "9")   // sell 1 USD at 9 KCN
	ask7 := h.mustPlaceOrder(t, seller, entity.SellSide, "1", "7")   // sell 1 USD at 7 KCN
	ask10 := h.mustPlaceOrder(t, seller, entity.SellSide, "1", "10") // sell 1 USD at 10 KCN: above the bid
	ask8 := h.mustPlaceOrder(t, seller, entity.SellSide, "1", "8")   // sell 1 USD at 8 KCN
	bidID := h.mustPlaceOrder(t, buyer, entity.BuySide, "4", "9")    // buy 4 USD at 9 KCN: 7, 8, 9, then rests

	assert.Equal(t, []tradeRow{
		{BuyOrderID: bidID, SellOrderID: ask7, Price: 7, Quantity: 100},
		{BuyOrderID: bidID, SellOrderID: ask8, Price: 8, Quantity: 100},
		{BuyOrderID: bidID, SellOrderID: ask9, Price: 9, Quantity: 100},
	}, h.trades(t))
	assert.Equal(t, orderRow{ID: ask7, Side: entity.SellSide, Price: 7, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, ask7))
	assert.Equal(t, orderRow{ID: ask8, Side: entity.SellSide, Price: 8, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, ask8))
	assert.Equal(t, orderRow{ID: ask9, Side: entity.SellSide, Price: 9, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, ask9))
	assert.Equal(t, orderRow{ID: ask10, Side: entity.SellSide, Price: 10, QuantityOrdered: 100,
		Status: entity.StatusOpen}, h.order(t, ask10))
	assert.Equal(t, orderRow{ID: bidID, Side: entity.BuySide, Price: 9, QuantityOrdered: 400,
		QuantityFilled: 300, Status: entity.StatusPartiallyFilled}, h.order(t, bidID))

	// The seller got 7 + 8 + 9 = 24 KCN; the ask at 10 still rests. The bid reserved
	// 4 * 9 = 36 KCN, paid 24, got back (9 - 7) + (9 - 8) = 3, and keeps 1 * 9 = 9 for
	// the 1 USD still resting.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 600, Reserved: 100}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 24}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 67, Reserved: 9}, buyerBalances.Quote)
}

func TestMatching_Ask_SweepsBidLevels(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	// Placed out of price order, so only price decides which fills first.
	bid7 := h.mustPlaceOrder(t, buyer, entity.BuySide, "1", "7")    // buy 1 USD at 7 KCN
	bid9 := h.mustPlaceOrder(t, buyer, entity.BuySide, "1", "9")    // buy 1 USD at 9 KCN
	bid6 := h.mustPlaceOrder(t, buyer, entity.BuySide, "1", "6")    // buy 1 USD at 6 KCN: below the ask
	bid8 := h.mustPlaceOrder(t, buyer, entity.BuySide, "1", "8")    // buy 1 USD at 8 KCN
	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "4", "7") // sell 4 USD at 7 KCN: 9, 8, 7, then rests

	assert.Equal(t, []tradeRow{
		{BuyOrderID: bid9, SellOrderID: askID, Price: 9, Quantity: 100},
		{BuyOrderID: bid8, SellOrderID: askID, Price: 8, Quantity: 100},
		{BuyOrderID: bid7, SellOrderID: askID, Price: 7, Quantity: 100},
	}, h.trades(t))
	assert.Equal(t, orderRow{ID: bid9, Side: entity.BuySide, Price: 9, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, bid9))
	assert.Equal(t, orderRow{ID: bid8, Side: entity.BuySide, Price: 8, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, bid8))
	assert.Equal(t, orderRow{ID: bid7, Side: entity.BuySide, Price: 7, QuantityOrdered: 100,
		QuantityFilled: 100, Status: entity.StatusFilled}, h.order(t, bid7))
	assert.Equal(t, orderRow{ID: bid6, Side: entity.BuySide, Price: 6, QuantityOrdered: 100,
		Status: entity.StatusOpen}, h.order(t, bid6))
	assert.Equal(t, orderRow{ID: askID, Side: entity.SellSide, Price: 7, QuantityOrdered: 400,
		QuantityFilled: 300, Status: entity.StatusPartiallyFilled}, h.order(t, askID))

	// The seller got 9 + 8 + 7 = 24 KCN and keeps 1 USD reserved for the rest of the ask.
	// The resting bids paid their own prices, so nothing comes back; the bid at 6 keeps
	// its 6 KCN.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 600, Reserved: 100}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 24}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 70, Reserved: 6}, buyerBalances.Quote)
}
