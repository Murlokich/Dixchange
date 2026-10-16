//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	cancelorder "github.com/dixchange/Monolith/internal/domain/usecase/cancel_order"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	getpairbalance "github.com/dixchange/Monolith/internal/domain/usecase/get_pair_balance"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
)

const (
	testPassword = "Passw0rd!"

	settleTimeout  = 5 * time.Second
	settlePollRate = 10 * time.Millisecond
)

// newUser registers a user and deposits the given amounts of the pair's base and quote
// symbols, as decimal strings ("" or "0" deposits nothing). It returns the user's ID.
func (h *harness) newUser(t *testing.T, base, quote string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	email := "user-" + uuid.NewString() + "@test.com"
	if _, err := h.register.Execute(ctx, register.Input{Email: email, Password: testPassword}); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Register returns nothing, so the ID is looked up by the unique email.
	var userID uuid.UUID
	if err := testDB.QueryRow(ctx, `SELECT id FROM auth.user WHERE email = $1`, email).Scan(&userID); err != nil {
		t.Fatalf("look up registered user: %v", err)
	}

	h.depositFunds(t, userID, h.pair.Base.ID, base)
	h.depositFunds(t, userID, h.pair.Quote.ID, quote)
	return userID
}

func (h *harness) depositFunds(t *testing.T, userID uuid.UUID, symbolID int, amount string) {
	t.Helper()
	if amount == "" {
		return
	}
	value := decimal.RequireFromString(amount)
	if value.IsZero() {
		return
	}
	_, err := h.deposit.Execute(context.Background(), deposit.Input{UserID: userID, SymbolID: symbolID, Amount: value})
	if err != nil {
		t.Fatalf("deposit %s of symbol %d: %v", amount, symbolID, err)
	}
}

// placeOrder calls the create order use case on the pair, with quantity and price as
// decimal strings. It returns once the command is accepted, before the engine has placed
// the order: call waitSettled before checking the result.
func (h *harness) placeOrder(t *testing.T, userID uuid.UUID, side, quantity, price string) error {
	t.Helper()
	_, err := h.createOrder.Execute(context.Background(), createorder.Input{
		UserID:          userID,
		MarketPairID:    pairID,
		Side:            side,
		QuantityOrdered: decimal.RequireFromString(quantity),
		PricePerUnit:    decimal.RequireFromString(price),
	})
	return err
}

// cancel calls the cancel order use case. It returns once the command is accepted, before
// the engine has removed the order: call waitSettled before checking the result.
func (h *harness) cancel(t *testing.T, userID, orderID uuid.UUID) error {
	t.Helper()
	_, err := h.cancelOrder.Execute(context.Background(), cancelorder.Input{UserID: userID, OrderID: orderID})
	return err
}

// waitSettled waits until every command accepted so far has gone through the engine and
// settlement. It needs both running: a stopped one never drains.
//
// Settled means the command log is empty and settlement has finished every event. The log
// alone isn't enough: settlement takes a place command before settling its trades. The
// log is checked first: once it is empty every event has been published, so the event
// counts can't grow anymore.
func (h *harness) waitSettled(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(settleTimeout)
	for {
		h.failIfStopped(t)

		pending := h.commandLogLen(t)
		if pending == 0 && h.eventBus.idle() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("not settled within %s: %d commands in the log, %d events published, %d finished",
				settleTimeout, pending, h.eventBus.published.Load(), h.eventBus.finished.Load())
		}
		time.Sleep(settlePollRate)
	}
}

func (h *harness) commandLogLen(t *testing.T) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(context.Background(), `SELECT count(*) FROM "order".command_log_usd_kcn`).Scan(&n); err != nil {
		t.Fatalf("count command log: %v", err)
	}
	return n
}

// balances returns the user's base and quote balances, in minor units.
func (h *harness) balances(t *testing.T, userID uuid.UUID) getpairbalance.Output {
	t.Helper()
	out, err := h.getPairBalance.Execute(context.Background(), getpairbalance.Input{UserID: userID, PairID: pairID})
	if err != nil {
		t.Fatalf("get pair balance: %v", err)
	}
	return out
}

// orderRow is an order as stored, in minor units.
type orderRow struct {
	ID              uuid.UUID
	Side            string
	Price           int64
	QuantityOrdered int64
	QuantityFilled  int64
	Status          string
}

// orders returns all the user's orders in any status, oldest first. It reads the table
// directly: the get orders use case returns active orders only.
func (h *harness) orders(t *testing.T, userID uuid.UUID) []orderRow {
	t.Helper()

	rows, err := testDB.Query(context.Background(), `
		SELECT id, side::text, price, quantity_ordered, quantity_filled, status
		FROM "order"."order" WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		t.Fatalf("query orders: %v", err)
	}
	defer rows.Close()

	var orders []orderRow
	for rows.Next() {
		var o orderRow
		if err = rows.Scan(&o.ID, &o.Side, &o.Price, &o.QuantityOrdered, &o.QuantityFilled, &o.Status); err != nil {
			t.Fatalf("scan order: %v", err)
		}
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("read orders: %v", err)
	}
	return orders
}
