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
- **DESIGN: create returns no order id, and a cancel works only after the order is settled.**
  Found 2026-10-21 while designing the randomized e2e suite. Not fixed yet (see "Why not now").

  *Problem, part 1: the create contract.* `create_order` returns an empty `Output` and the
  handler answers 202 without a body, so a client learns its order's id only by polling
  "my orders" until the order is settled. Every client that acts on its own orders needs the id
  right away: automated trading, k6 scripts, the Phase 2 black-box suite. The id is generated
  in the use case before anything is written, so returning it is trivial.

  *Problem, part 2: cancel needs the settled order.* The cancel use case checks ownership and
  status with `GetOrderCancelInfo`, which reads `orders`, the post-engine snapshot. An order
  that is accepted but not yet settled exists only as a place row in the command log, so the
  cancel gets `ErrOrderNotFound`, the handler answers the fake 202, and the cancel is silently
  dropped. Today part 1 hides part 2: a client can't know the id before settlement. Returning
  the id makes it reachable: place and cancel right away, and the cancel is lost without the
  client being told.

  *Why it is wrong.* Ownership is decided when the order is accepted and never changes; only
  the status depends on the engine. Reading both from the post-engine snapshot makes ownership
  visible too late. Desired: a cancel of an accepted but not yet applied order is validated
  (ownership) and accepted. Ordering is already safe for it: the place command is sent before
  the create request returns, the cancel goes later through the same FIFO command bus, so the
  engine applies place, then cancel. On recovery both are in the log and the cancel's id is the
  later one, so the replay keeps that order.

  *Rejected:*
  - Skip the ownership check when the order isn't in `orders`: anyone could cancel anyone's
    pending order.
  - Validate in the engine (user id in the cancel command and in the book): the book doesn't
    know users and its contract stays that way; it also breaks the trust boundary (the order
    service checks ownership, the matching engine trusts the commands it gets).

  *Candidate fix: read ownership where the order was accepted.* Look the order up in `orders`
  or, if it isn't there yet, in the command log, whose place row holds `user_id` (in the
  `place` JSONB). An order found only in the log is pending, which counts as cancellable.
  - **Race:** two separate queries are wrong. Settlement can move the order from the log into
    `orders` between them, the order is found in neither, and a valid cancel gets
    `ErrOrderNotFound`. The lookup must be one SQL statement (e.g. a `UNION` of both lookups:
    one statement reads one snapshot under READ COMMITTED) or run in a REPEATABLE READ tx.
  - **Phase 2:** the command log becomes an outbox / broker topic, which can't be queried.
    That doesn't break the fix: ownership never needs the broker. The order service keeps its
    own queryable record of accepted orders (it needs one anyway, for "my orders" to show
    pending orders), e.g. the order row inserted at acceptance with a `pending` status, or an
    ownership table. Only where the record lives changes, not the rule. The same `pending` row
    would work in the monolith too, but it is a bigger change: settlement would update the
    row instead of inserting it, and recovery would have to skip pending rows.

  *Alternative: make it the contract.* "An order can be cancelled once it is live; before that,
  cancel answers not found." Valid as eventual consistency, but then the cancel of one's own
  pending order must return a real error instead of the fake 202 (the fake 202 only hides
  other users' orders), clients need retry logic, and tests have to accept "not found" for
  their own orders, which hides real not-found bugs.

  *Why not now (2026-10-21).* Phase 1 is past its deadline. Nothing can reach part 2 while
  create returns no id, so current behaviour is consistent, only limited. The ownership record
  moves anyway in Phase 2 (order service with its own database), so the fix is best designed
  there. Workaround in the e2e tests: order ids are read from `orders` after settlement
  (`mustPlaceOrder`; the randomized suite cancels only orders it has seen in `orders`).

  *Where it will come up again.* k6: a scenario that places and cancels needs ids, so it must
  poll, and a place-then-cancel scenario would see silent drops. The Phase 2 black-box suite:
  same, over HTTP. Microservices: create in the order service and an engine behind a broker
  make the window between acceptance and settlement longer.

  *Steps when picked up.* (1) Create returns the order id: use case `Output`, 202 response
  body, their unit tests. (2) Ownership lookup in one statement over `orders` + the log (or
  the Phase 2 record). (3) Keep the fake 202 for other users' orders. (4) e2e: drop the id
  lookup in `mustPlaceOrder`; add a test: engine stopped, place, cancel right away -> both
  accepted, the order ends cancelled with its funds released.
