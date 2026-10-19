//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

// After a crash the app rebuilds the book from the database: the active orders, then the
// command log replayed through the engine. A command accepted before the crash is in the
// log, with its funds reserved, even if the engine never got it: the restart places it
// once, without reserving its funds again.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestLoadCommands_EngineStopped_ReplaysLog(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "", "100") // no USD, 100 KCN

	// With the engine stopped the orders reach only the command log and the command bus,
	// which the crash loses. Bids that don't cross: 5 + 8 + 9 = 22 KCN reserved.
	h.stopEngine(t)
	for _, o := range []struct{ quantity, price string }{{"1", "5"}, {"2", "4"}, {"3", "3"}} {
		err := h.placeOrder(t, user, entity.BuySide, o.quantity, o.price)
		require.NoError(t, err)
	}

	assert.Empty(t, h.orders(t, user))
	assert.Equal(t, 3, h.commandLogLen(t), "command log")
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 78, Reserved: 22}, balances.Quote)

	h.restart(t)
	h.waitSettled(t)

	// The orders are placed in the order they were accepted, each once.
	orders := h.orders(t, user)
	require.Len(t, orders, 3)
	assert.Equal(t, []orderRow{
		{ID: orders[0].ID, Side: entity.BuySide, Price: 5, QuantityOrdered: 100, Status: entity.StatusOpen},
		{ID: orders[1].ID, Side: entity.BuySide, Price: 4, QuantityOrdered: 200, Status: entity.StatusOpen},
		{ID: orders[2].ID, Side: entity.BuySide, Price: 3, QuantityOrdered: 300, Status: entity.StatusOpen},
	}, orders)
	assert.Zero(t, h.commandLogLen(t), "command log")
	balances = h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: 78, Reserved: 22}, balances.Quote)
}
