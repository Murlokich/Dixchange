//go:build e2e

package e2e

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

// The command bus admits a command only while it has room for it, before anything is
// written: with the engine stopped nothing leaves the bus, so it takes exactly its
// capacity and rejects every command after that as overloaded, with no funds reserved and
// no command logged. Once the engine runs again, the admitted commands are placed and the
// bus takes commands again.
//
// Amounts are in minor units: the pair is USD/KCN, base USD with 2 decimals (1 USD = 100),
// quote KCN with none.

func TestCommandBus_Full_RejectsOverload(t *testing.T) {
	h := newHarness(t)
	capacity := h.cfg.CommandBusConfig.BufferSize
	const overflow = 3

	// Each bid of 1 USD at 1 KCN reserves 1 KCN. The user can pay for every attempt, so a
	// rejection can only come from the bus.
	user := h.newUser(t, "", strconv.Itoa(capacity+overflow))

	h.stopEngine(t)
	for i := range capacity {
		err := h.placeOrder(t, user, entity.BuySide, "1", "1")
		require.NoError(t, err, "order %d of %d", i+1, capacity)
	}
	for i := range overflow {
		err := h.placeOrder(t, user, entity.BuySide, "1", "1")
		require.ErrorIs(t, err, entity.ErrCommandBusOverloaded, "order %d over capacity", i+1)
	}

	// Only the admitted orders reserved funds and were logged.
	balances := h.balances(t, user)
	assert.Equal(t, entity.Balance{SymbolID: h.pair.Quote.ID, Main: overflow, Reserved: int64(capacity)}, balances.Quote)
	assert.Equal(t, capacity, h.commandLogLen(t), "command log")

	h.startEngine()
	h.waitSettled(t)

	// Every admitted order rests in the book: nothing crosses, all are bids.
	open := 0
	for _, o := range h.orders(t, user) {
		if o.Status == entity.StatusOpen {
			open++
		}
	}
	assert.Equal(t, capacity, open, "open orders")

	// The engine freed the slots, so the bus takes commands again.
	err := h.placeOrder(t, user, entity.BuySide, "1", "1")
	require.NoError(t, err)
}
