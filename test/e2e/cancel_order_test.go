//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dixchange/Monolith/internal/domain/entity"
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

	err := h.placeOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	require.NoError(t, err)
	h.waitSettled(t)
	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	orderID := orders[0].ID

	err = h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.waitSettled(t)

	// The 3 USD reserved by the ask are available again.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)

	orders = h.orders(t, user)
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

	err := h.placeOrder(t, user, entity.BuySide, "3", "7") // buy 3 USD at 7 KCN
	require.NoError(t, err)
	h.waitSettled(t)
	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	orderID := orders[0].ID

	err = h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.waitSettled(t)

	// The 21 KCN reserved by the bid are available again.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 100}, balances.Quote)

	orders = h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.BuySide,
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

	err := h.placeOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	require.NoError(t, err)
	h.waitSettled(t)
	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	orderID := orders[0].ID

	// With the engine stopped, the first cancel can't be settled before the second one
	// checks the order: both wait in the command bus.
	h.stopEngine(t)
	err = h.cancel(t, user, orderID)
	require.NoError(t, err)
	err = h.cancel(t, user, orderID)
	require.NoError(t, err)
	h.startEngine()
	h.waitSettled(t)

	// The 3 USD reserved by the ask are available again, released once.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 1000}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)

	orders = h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orderID,
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusCancelled,
	}, orders[0])
}
