//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

// funds is an amount of the pair's base and quote, in minor units.
type funds struct {
	Base  int64
	Quote int64
}

// checkInvariants checks the rules the state must follow whatever happened before, reading
// the database directly. It is meant for a settled state (see waitSettled): with the
// command log empty, the active orders are exactly the book. The deposits are the ones
// depositFunds recorded. Every violation is reported, not only the first. newHarness runs
// it when the test ends; a test may also call it on a settled state of its own.
func (h *harness) checkInvariants(t *testing.T) {
	t.Helper()
	s := h.readState(t)

	h.checkLedger(t, s, h.deposits)
	h.checkReservations(t, s)
	checkTrades(t, s)
	checkOrders(t, s)
	checkBookNotCrossed(t, s)
	checkBalancesNotNegative(t, s)
}

// state is everything the invariants look at. Users without a balance row of a symbol
// have the zero balance there.
type state struct {
	base   map[uuid.UUID]entity.Balance
	quote  map[uuid.UUID]entity.Balance
	orders map[uuid.UUID]stateOrder
	trades []stateTrade
}

type stateOrder struct {
	orderRow
	UserID uuid.UUID
}

type stateTrade struct {
	tradeRow
	BuyerID  uuid.UUID
	SellerID uuid.UUID
}

// remaining is what the order hasn't filled yet.
func (o stateOrder) remaining() int64 {
	return o.QuantityOrdered - o.QuantityFilled
}

// users returns every user the state or the deposits know about.
func (s state) users(deposits map[uuid.UUID]funds) map[uuid.UUID]bool {
	users := make(map[uuid.UUID]bool)
	for id := range deposits {
		users[id] = true
	}
	for id := range s.base {
		users[id] = true
	}
	for id := range s.quote {
		users[id] = true
	}
	for _, o := range s.orders {
		users[o.UserID] = true
	}
	return users
}

// Recalculates each user's funds from their deposits and the trades they made, then
// compares the result with their balances.
func (h *harness) checkLedger(t *testing.T, s state, deposits map[uuid.UUID]funds) {
	t.Helper()

	expected := make(map[uuid.UUID]funds)
	for id, d := range deposits {
		expected[id] = d
	}
	for _, tr := range s.trades {
		notional, err := h.pair.Notional(tr.Quantity, tr.Price)
		if err != nil {
			t.Errorf("ledger: notional of trade %+v: %v", tr.tradeRow, err)
			continue
		}
		buyer, seller := expected[tr.BuyerID], expected[tr.SellerID]
		buyer.Base += tr.Quantity
		buyer.Quote -= notional
		expected[tr.BuyerID] = buyer
		seller.Base -= tr.Quantity
		seller.Quote += notional
		expected[tr.SellerID] = seller
	}

	for id := range s.users(deposits) {
		base, quote := s.base[id], s.quote[id]
		held := funds{Base: base.Main + base.Reserved, Quote: quote.Main + quote.Reserved}
		if held != expected[id] {
			t.Errorf("ledger: user %s holds %+v, deposits and trades give %+v", id, held, expected[id])
		}
	}
}

// Recalculates the expected reserved funds from the active (open and partially filled)
// orders, then compares them with the balances.
func (h *harness) checkReservations(t *testing.T, s state) {
	t.Helper()

	expected := make(map[uuid.UUID]funds)
	for _, o := range s.orders {
		if !entity.IsActiveOrderStatus(o.Status) {
			continue
		}
		reserved := expected[o.UserID]
		if o.Side == entity.SellSide {
			reserved.Base += o.remaining()
		} else {
			notional, err := h.pair.Notional(o.remaining(), o.Price)
			if err != nil {
				t.Errorf("reservations: notional of order %+v: %v", o.orderRow, err)
				continue
			}
			reserved.Quote += notional
		}
		expected[o.UserID] = reserved
	}

	for id := range s.users(nil) {
		reserved := funds{Base: s.base[id].Reserved, Quote: s.quote[id].Reserved}
		if reserved != expected[id] {
			t.Errorf("reservations: user %s has %+v reserved, active orders need %+v", id, reserved, expected[id])
		}
	}
}

// Recalculates each order's filled quantity from its trades, then compares it with the
// order's QuantityFilled.
func checkTrades(t *testing.T, s state) {
	t.Helper()

	traded := make(map[uuid.UUID]int64)
	for _, tr := range s.trades {
		traded[tr.BuyOrderID] += tr.Quantity
		traded[tr.SellOrderID] += tr.Quantity
		checkTrade(t, s, tr)
	}
	for id, o := range s.orders {
		if traded[id] != o.QuantityFilled {
			t.Errorf("trades: order %s filled %d, its trades add up to %d", id, o.QuantityFilled, traded[id])
		}
	}
}

// Validates a trade's basic data: the orders it refers to, its quantity, its price etc.
func checkTrade(t *testing.T, s state, tr stateTrade) {
	t.Helper()

	bid, bidOK := s.orders[tr.BuyOrderID]
	ask, askOK := s.orders[tr.SellOrderID]
	switch {
	case !bidOK || !askOK:
		t.Errorf("trades: trade %+v refers to a missing order", tr.tradeRow)
	case bid.Side != entity.BuySide || ask.Side != entity.SellSide:
		t.Errorf("trades: trade %+v isn't between a bid and an ask: %s, %s", tr.tradeRow, bid.Side, ask.Side)
	case bid.UserID != tr.BuyerID || ask.UserID != tr.SellerID:
		t.Errorf("trades: trade %+v names buyer %s and seller %s, the orders belong to %s and %s",
			tr.tradeRow, tr.BuyerID, tr.SellerID, bid.UserID, ask.UserID)
	case tr.Quantity <= 0:
		t.Errorf("trades: trade %+v has no quantity", tr.tradeRow)
	case tr.Price > bid.Price || tr.Price < ask.Price:
		t.Errorf("trades: trade %+v is outside the bid %d and the ask %d", tr.tradeRow, bid.Price, ask.Price)
	case tr.Price != bid.Price && tr.Price != ask.Price:
		t.Errorf("trades: trade %+v is at neither the bid %d nor the ask %d", tr.tradeRow, bid.Price, ask.Price)
	}
}

// Validates that an order's status matches its filled quantity:
// Open -> QuantityFilled == 0
// PartiallyFilled -> 0 < QuantityFilled < QuantityOrdered
// Filled -> QuantityFilled == QuantityOrdered
// Cancelled -> QuantityFilled != QuantityOrdered
func checkOrders(t *testing.T, s state) {
	t.Helper()

	for id, o := range s.orders {
		if o.QuantityOrdered <= 0 || o.QuantityFilled < 0 || o.QuantityFilled > o.QuantityOrdered {
			t.Errorf("orders: order %s filled %d of %d", id, o.QuantityFilled, o.QuantityOrdered)
			continue
		}
		var ok bool
		switch o.Status {
		case entity.StatusOpen:
			ok = o.QuantityFilled == 0
		case entity.StatusPartiallyFilled:
			ok = o.QuantityFilled > 0 && o.QuantityFilled < o.QuantityOrdered
		case entity.StatusFilled:
			ok = o.QuantityFilled == o.QuantityOrdered
		case entity.StatusCancelled:
			ok = o.QuantityFilled < o.QuantityOrdered
		}
		if !ok {
			t.Errorf("orders: order %s is %s with %d of %d filled", id, o.Status, o.QuantityFilled, o.QuantityOrdered)
		}
	}
}

// The book isn't crossed: every active bid is below every active ask, or they would
// have traded.
func checkBookNotCrossed(t *testing.T, s state) {
	t.Helper()

	var bestBid, bestAsk *stateOrder
	for _, o := range s.orders {
		if !entity.IsActiveOrderStatus(o.Status) {
			continue
		}
		if o.Side == entity.BuySide && (bestBid == nil || o.Price > bestBid.Price) {
			bestBid = &o
		}
		if o.Side == entity.SellSide && (bestAsk == nil || o.Price < bestAsk.Price) {
			bestAsk = &o
		}
	}
	if bestBid != nil && bestAsk != nil && bestBid.Price >= bestAsk.Price {
		t.Errorf("book: crossed, bid %s at %d, ask %s at %d", bestBid.ID, bestBid.Price, bestAsk.ID, bestAsk.Price)
	}
}

// No balance is negative.
func checkBalancesNotNegative(t *testing.T, s state) {
	t.Helper()

	for _, balances := range []map[uuid.UUID]entity.Balance{s.base, s.quote} {
		for id, b := range balances {
			if b.Main < 0 || b.Reserved < 0 {
				t.Errorf("balances: user %s has %+v", id, b)
			}
		}
	}
}

// readState reads the balances of the pair's symbols, every order and every trade.
func (h *harness) readState(t *testing.T) state {
	t.Helper()
	return state{
		base:   readBalances(t, h.pair.Base.ID),
		quote:  readBalances(t, h.pair.Quote.ID),
		orders: readOrders(t),
		trades: readTrades(t),
	}
}

func readBalances(t *testing.T, symbolID int) map[uuid.UUID]entity.Balance {
	t.Helper()

	rows, err := testDB.Query(context.Background(),
		`SELECT user_id, main, reserved FROM wallet.balance WHERE symbol_id = $1`, symbolID)
	if err != nil {
		t.Fatalf("query balances: %v", err)
	}
	defer rows.Close()

	balances := make(map[uuid.UUID]entity.Balance)
	for rows.Next() {
		var userID uuid.UUID
		b := entity.Balance{SymbolID: symbolID}
		if err = rows.Scan(&userID, &b.Main, &b.Reserved); err != nil {
			t.Fatalf("scan balance: %v", err)
		}
		balances[userID] = b
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("read balances: %v", err)
	}
	return balances
}

func readOrders(t *testing.T) map[uuid.UUID]stateOrder {
	t.Helper()

	rows, err := testDB.Query(context.Background(), `
		SELECT id, user_id, side::text, price, quantity_ordered, quantity_filled, status
		FROM "order"."order"`)
	if err != nil {
		t.Fatalf("query orders: %v", err)
	}
	defer rows.Close()

	orders := make(map[uuid.UUID]stateOrder)
	for rows.Next() {
		var o stateOrder
		err = rows.Scan(&o.ID, &o.UserID, &o.Side, &o.Price, &o.QuantityOrdered, &o.QuantityFilled, &o.Status)
		if err != nil {
			t.Fatalf("scan order: %v", err)
		}
		orders[o.ID] = o
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("read orders: %v", err)
	}
	return orders
}

func readTrades(t *testing.T) []stateTrade {
	t.Helper()

	rows, err := testDB.Query(context.Background(), `
		SELECT buy_order_id, sell_order_id, price, quantity, buyer_id, seller_id
		FROM trade.trade ORDER BY id`)
	if err != nil {
		t.Fatalf("query trades: %v", err)
	}
	defer rows.Close()

	var trades []stateTrade
	for rows.Next() {
		var tr stateTrade
		err = rows.Scan(&tr.BuyOrderID, &tr.SellOrderID, &tr.Price, &tr.Quantity, &tr.BuyerID, &tr.SellerID)
		if err != nil {
			t.Fatalf("scan trade: %v", err)
		}
		trades = append(trades, tr)
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("read trades: %v", err)
	}
	return trades
}
