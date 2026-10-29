//go:build e2e

package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
)

// A cancelled order leaves the book and its funds go back to the user: once settled, the
// order is stored as cancelled and nothing stays reserved for it. A bid and an ask reserve
// different symbols, so each side has its own test.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestCancelOrder_Ask_ReleasesBaseQuantity(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "10", "") // 10 USD, no KCN

	orderID := h.mustPlaceOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN

	err := h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.waitSettled(t)

	// The 3 USD reserved by the ask are available again.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusCancelled,
	}, orders[0])
}

func TestCancelOrder_Bid_ReleasesQuoteNotional(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "", "100") // no USD, 100 KCN

	orderID := h.mustPlaceOrder(t, user, entity.BuySide, "3", "7") // buy 3 USD at 7 KCN

	err := h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.waitSettled(t)

	// The 21 KCN reserved by the bid are available again.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 100}, balances.Quote)

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.BuySide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusCancelled,
	}, orders[0])
}

// A cancel of an order that is already cancelled is rejected in the request: the order
// isn't active in the DB anymore, so no command is written and nothing changes.

func TestCancelOrder_AlreadyCancelled_Rejected(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "10", "") // 10 USD, no KCN

	orderID := h.mustPlaceOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN

	err := h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.waitSettled(t)

	err = h.cancel(t, user, orderID)
	require.ErrorIs(t, err, entity.ErrOrderNotCancellable)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	// Released by the first cancel only.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusCancelled,
	}, orders[0])
}

// Two cancels of the same order can both be accepted: each checks the order in the DB,
// where it stays active until the first cancel is settled. The engine removes the order
// on the first and rejects the second, so the funds are released only once.

func TestCancelOrder_Twice_ReleasesFundsOnce(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "10", "") // 10 USD, no KCN

	orderID := h.mustPlaceOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN

	// With the engine stopped, the first cancel can't be settled before the second one
	// checks the order: both wait in the command bus.
	h.stopEngine(t)
	err := h.cancel(t, user, orderID)
	require.NoError(t, err)
	err = h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.startEngine()
	h.waitSettled(t)

	// The 3 USD reserved by the ask are available again, released once.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusCancelled,
	}, orders[0])
}

// A cancel after a trade releases only what is still reserved for the unfilled part: the
// filled part was already paid to the other side. An order that is filled completely is
// no longer active, so its cancel is rejected in the request.

func TestCancelOrder_PartiallyFilled_ReleasesRemainder(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	h.mustPlaceOrder(t, buyer, entity.BuySide, "1", "7")            // buy 1 USD at 7 KCN: fills 1 of the 3

	err := h.cancel(t, seller, askID)
	require.NoError(t, err)
	h.waitSettled(t)

	// The seller sold 1 USD for 7 KCN; the 2 USD still reserved are available again.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 900}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 7}, sellerBalances.Quote)
	// The buyer is untouched by the cancel: 1 USD bought for 7 KCN.
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 100}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 93}, buyerBalances.Quote)

	orders := h.orders(t, seller)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              askID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		QuantityFilled:  100,
		Status:          entity.StatusCancelled,
	}, orders[0])
}

func TestCancelOrder_Filled_Rejected(t *testing.T) {
	h := newHarness(t)
	seller := h.newUser(t, "10", "") // 10 USD, no KCN
	buyer := h.newUser(t, "", "100") // no USD, 100 KCN

	askID := h.mustPlaceOrder(t, seller, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	h.mustPlaceOrder(t, buyer, entity.BuySide, "3", "7")            // buy 3 USD at 7 KCN: fills all 3

	err := h.cancel(t, seller, askID)
	require.ErrorIs(t, err, entity.ErrOrderNotCancellable)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	// Only the trade moved funds: the seller sold 3 USD for 21 KCN, nothing is reserved.
	sellerBalances := h.balances(t, seller)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700}, sellerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 21}, sellerBalances.Quote)
	buyerBalances := h.balances(t, buyer)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 300}, buyerBalances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 79}, buyerBalances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")

	orders := h.orders(t, seller)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              askID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		QuantityFilled:  300,
		Status:          entity.StatusFilled,
	}, orders[0])
}

// A cancel of an order that doesn't exist is rejected in the request: no command is
// written and the user's funds stay as they were.

func TestCancelOrder_UnknownOrder_NotFound(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "10", "") // 10 USD, no KCN

	err := h.cancel(t, user, uuid.New())
	require.ErrorIs(t, err, entity.ErrOrderNotFound)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")
}

// A cancel of another user's order is rejected in the request, before a command is
// written: the order stays in the book and its funds stay reserved.

func TestCancelOrder_OtherUsersOrder_NotOwned(t *testing.T) {
	h := newHarness(t)
	owner := h.newUser(t, "10", "") // 10 USD, no KCN
	other := h.newUser(t, "", "")

	orderID := h.mustPlaceOrder(t, owner, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN

	err := h.cancel(t, other, orderID)
	require.ErrorIs(t, err, entity.ErrOrderNotOwned)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	// The 3 USD stay reserved for the owner's order.
	balances := h.balances(t, owner)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700, Reserved: 300}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")

	orders := h.orders(t, owner)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusOpen,
	}, orders[0])
}
