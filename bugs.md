# Bugs and open issues

Found 2026-10-11 while reviewing the known gaps. Fix after the critical-path tests;
add new findings here instead of fixing them mid-task.

- **BUG: crossing snapshot -> crash loop.** Place settlement inserts the order in tx 1, then
  settles each trade in its own tx. A crash or trade failure between them leaves the new order
  and its counterparties active in `orders` with unfilled quantities that cross. On startup
  `loadOpenOrders` inserts resting orders, they match, and it fails with "unexpected trade
  produced by resting orders" -> restart -> same failure forever. This is the exact case the
  design says converges ("unsettled trades are re-matched"). Options: settle the order + all its
  trades in one tx (removes the window), or let recovery match the snapshot and settle the trades
  it produces.
- **BUG: startup hangs when the command log is longer than the event bus buffer.**
  `cmd/app.go` calls `LoadOrderBook` before settlement starts. Replaying the log publishes one
  event per command, and with nobody consuming, `publish` blocks once the event bus buffer (100)
  is full. The ctx is the signal ctx, so it waits forever: no crash, no restart, just a hung app.
  After a crash under load the log can hold up to command bus + event bus capacity (~200)
  commands, so this is reachable exactly when recovery matters. `recovery.md` already describes
  the right order (start engine and settlement, then replay); the code differs. Fix: start
  settlement before replaying the log (or replay through the command bus once the engine runs).
- **Poison command -> crash loop.** A command whose settlement fails deterministically (e.g. a
  constraint violation) fails again on every replay, so the exchange stays down. Accepted for
  the monolith; state it in the thesis (alternative: dead-letter the command).
- `ticket.Send*` failing after commit (`ErrAccessQueueAsync`, an invariant violation) returns a
  500 but doesn't crash: the command stays in the log with funds reserved until the next restart.
  Consider crashing, consistent with the failure policy.
- Ordering: `GetAllOpenOrders` sorts by `created_at` (settlement time = engine order), while
  `recovery.md` says id. `created_at` is the more correct one: ids are generated before the bus
  slot is taken, concurrently, so id order isn't engine order. Fix the docs, not the query.
  Needs the partial index from `recovery.md` and a pair filter once there are several books.
- `GetOrderCancelInfo` doesn't return `pair_id`; needed to route a cancel to the right book (and
  its command log table) once there is more than one book.
- Cancel returns 202 for a request, not a result; the frontend must poll orders to see it.
- Create returns no order id, so the client can't cancel an order it just placed until it shows
  up in "my orders".
