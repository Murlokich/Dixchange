//go:build e2e

package e2e

import (
	"strconv"
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

// A command the engine already placed is lost with the book if settlement hasn't applied
// it: its event dies in the event bus. Its log row is still there, so the restart places it
// again, once.

func TestLoadCommands_SettlementStopped_ReplaysLog(t *testing.T) {
	h := newHarness(t)
	user := h.newUser(t, "", "100") // no USD, 100 KCN

	// With settlement stopped the engine places the orders, but their events only reach the
	// event bus, which the crash loses. Bids that don't cross: 5 + 8 + 9 = 22 KCN reserved.
	h.stopSettlement(t)
	for _, o := range []struct{ quantity, price string }{{"1", "5"}, {"2", "4"}, {"3", "3"}} {
		err := h.placeOrder(t, user, entity.BuySide, o.quantity, o.price)
		require.NoError(t, err)
	}
	// Every order has gone through the engine, so the crash loses them on the way out.
	h.waitPublished(t, 3)

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

// With settlement stopped the engine keeps placing orders until the event bus is full, so
// the log can hold more commands than the event bus: their events are lost in the crash.
// Replaying them publishes more events than the bus holds, which works only because
// settlement runs during the load and empties it.

func TestLoadCommands_SettlementStopped_ReplaysMoreThanEventBus(t *testing.T) {
	h := newHarness(t)
	commands := h.cfg.EventBusConfig.BufferSize + 1

	// Each bid of 1 USD at 1 KCN reserves 1 KCN; bids don't cross.
	user := h.newUser(t, "", strconv.Itoa(commands))

	// The engine places the first orders until the event bus is full and holds the last
	// one, waiting to publish its event.
	h.stopSettlement(t)
	for i := range commands {
		err := h.placeOrder(t, user, entity.BuySide, "1", "1")
		require.NoError(t, err, "order %d of %d", i+1, commands)
	}

	assert.Empty(t, h.orders(t, user))
	assert.Equal(t, commands, h.commandLogLen(t), "command log")
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Reserved: int64(commands)}, balances.Quote)

	h.restart(t)
	h.waitSettled(t)

	open := 0
	for _, o := range h.orders(t, user) {
		if o.Status == entity.StatusOpen {
			open++
		}
	}
	assert.Equal(t, commands, open, "open orders")
	assert.Zero(t, h.commandLogLen(t), "command log")
	balances = h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Reserved: int64(commands)}, balances.Quote)
}
