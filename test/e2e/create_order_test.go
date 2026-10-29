//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
)

// An order with enough funds reserves what it may spend right away, in the request, and
// reaches the orders once the engine has placed it. A bid and an ask reserve different
// symbols, so each side has its own test. The book is empty, so the order rests unmatched.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestCreateOrder_Ask_ReservesBaseQuantity(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "10", "") // 10 USD, no KCN

	err := h.placeOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	require.NoError(t, err)
	h.waitSettled(t)

	// An ask reserves the base it sells: 3 USD.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 700, Reserved: 300}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orders[0].ID, // generated, can't be known in advance
		Side:            entity.SellSide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusOpen,
	}, orders[0])
}

func TestCreateOrder_Bid_ReservesQuoteNotional(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "", "100") // no USD, 100 KCN

	err := h.placeOrder(t, user, entity.BuySide, "3", "7") // buy 3 USD at 7 KCN
	require.NoError(t, err)
	h.waitSettled(t)

	// A bid reserves the quote it pays at its own price: 3 * 7 = 21 KCN.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 79, Reserved: 21}, balances.Quote)

	orders := h.orders(t, user)
	require.Len(t, orders, 1)
	assert.Equal(t, orderRow{
		ID:              orders[0].ID, // generated, can't be known in advance
		Side:            entity.BuySide,
		Price:           7,
		QuantityOrdered: 300,
		Status:          entity.StatusOpen,
	}, orders[0])
}

// An order without enough funds is rejected in the request: the funds reservation and the
// command are one transaction, so nothing is reserved and nothing reaches the engine.

func TestCreateOrder_Ask_InsufficientBase_Rejected(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "2", "") // 2 USD, no KCN

	err := h.placeOrder(t, user, entity.SellSide, "3", "7") // sell 3 USD at 7 KCN
	require.ErrorIs(t, err, entity.ErrInsufficientBalance)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID, Main: 200}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID}, balances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")
	assert.Empty(t, h.orders(t, user))
}

func TestCreateOrder_Bid_InsufficientQuote_Rejected(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "", "100") // no USD, 100 KCN

	// Buying 3 USD at 50 KCN costs 3 * 50 = 150 KCN.
	err := h.placeOrder(t, user, entity.BuySide, "3", "50")
	require.ErrorIs(t, err, entity.ErrInsufficientBalance)
	// Returns at once if nothing was accepted; otherwise waits, and the checks below fail.
	h.waitSettled(t)

	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Base.ID}, balances.Base)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 100}, balances.Quote)
	assert.Zero(t, h.commandLogLen(t), "command log")
	assert.Empty(t, h.orders(t, user))
}
