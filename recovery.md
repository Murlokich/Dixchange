# Recovery

How the matching engine gets its order book back after a restart, and where this can be improved.

## Why it is needed

- The order book lives only in memory. When the process stops, the book is gone.
- The database still holds everything needed to rebuild it:
  - `"order"."order"`: every order whose placement has been settled, with its filled quantity. This is the **snapshot**.
  - `"order".command_log_<pair>`: the commands (place / cancel) accepted for a book whose effect hasn't reached the orders yet. This is the **log tail**.
- State of a book = snapshot + log tail, replayed in order. This is the same idea a database uses with its write-ahead log, and an event-sourced system uses with snapshots.

### The rule that makes it correct

- A log row is deleted in the same transaction that applies its effect to the orders. A row is never deleted while its effect exists only in memory.
- Rows can't be deleted one by one as "done" otherwise: a command changes other orders too (a trade reduces the resting order's remaining quantity), so only the snapshot can cover it.
- A place command is taken from the log in the transaction that inserts its order, before its trades are settled. If it were taken after the trades, a failure in between would leave the order both in the orders (partly filled) and in the log, and a replay would place it twice.

## Startup recovery

Done once, before the application accepts requests.

1. **Load resting orders.** Fetch the active orders (`open`, `partially_filled`) of the book's pair, ordered by `id`, and insert them straight into the book.
2. **Start the engine and settlement.**
3. **Replay the log.** Send the log rows, ordered by `id`, through the command bus like any new command.
4. **Start the HTTP server.**

### Details that matter

- **Order of loading is time priority.** Within a price level the book is FIFO, so orders must be inserted in creation order. Without `ORDER BY id` the database returns rows in no defined order. IDs are UUIDv7, so ID order is creation order.
- **Insert the remaining quantity, not the ordered one.** A partially filled order goes back with `quantity_ordered - quantity_filled`.
- **Resting orders don't go through the engine.** Their placement is already settled. Through the engine they would publish events, and settlement would insert them a second time.
- **Resting orders never cross.** The book would have matched them when they arrived. If inserting one produces a trade, the snapshot is inconsistent and startup must fail.
- **Log rows do go through the engine.** Their effect isn't in the orders yet, so they must produce events and be settled, exactly as when they first arrived.
- **HTTP starts last.** A new request arriving during the replay could get in between two old commands and break their order.

### Behaviour after a crash

- **Trades that were matched but not settled are matched again.** Their orders come back from the snapshot with the quantity they had before those trades. The result converges to a valid state, but not necessarily the same trades as before the crash.
- **Cancels that weren't settled are kept.** The cancel is still in the log and is replayed.
- **Commands the engine never received are kept.** E.g. a send that timed out after the commit. They are in the log and get placed on replay.

## Runtime reload (later)

The same rebuild while the application runs, triggered by any failure that leaves the book and the database diverged. The places that need it are marked `TODO(recovery)` in the code. On top of startup recovery it needs:

- pausing new commands while the book is rebuilt;
- draining the command queue, since its commands are in the log anyway;
- a barrier so a request can't commit a log row between "the reload read the log" and "the reload finished", otherwise its command is lost or applied twice.

## Performance and future improvements

### Sizing

A pessimistic case: 30K bot users with 100 open orders each is 3M active orders. `entity.Order` is about 144 bytes, so holding them all at once costs about 0.5 GB before any overhead. That buffer is temporary; what stays is the book itself (an `orderbook.Order` of about 80 bytes plus a list node and a map entry per order).

### Index for the recovery query

- The query filters on two statuses and sorts by `id`. Without an index it scans the whole orders table, which keeps growing with filled and cancelled history.
- A **partial index** on `(market_pair_id, id)` covering only `status IN ('open', 'partially_filled')`:
  - holds only active orders, so it stays small as history grows;
  - matches both the filter and the sort, so no separate sort step;
  - works better than a plain index on `status`, which has only four values and is often skipped by the planner.

### Status as enum instead of text

- `status` is text with a `CHECK` constraint. A Postgres enum is smaller (4 bytes) and faster to compare.
- With the partial index the status condition is part of the index definition, so it barely matters for recovery. Worth a benchmark later, together with the other queries that filter by status.

### Streaming the rows instead of loading them

- sqlc's `:many` collects the whole result into a slice before returning it. At 3M rows that is the 0.5 GB buffer above, and the book can't start filling until the last row arrives.
- pgx already receives rows one by one (binary protocol, read from the socket incrementally). A hand-written loop over `rows.Next()` can insert each order into the book as it is read:
  - no buffer of the whole result;
  - fetching and inserting overlap, without any extra goroutine.
- `COPY ... TO STDOUT` isn't needed for this: a plain `SELECT` is already streamed.

### Other bottlenecks to measure

- **Inserting into the book is single-threaded.** The per-order insert cost decides recovery time at millions of orders.
- **The log replay settles one transaction per command.** A long log means many small commits.
- **Startup time grows with the number of active orders.** If it becomes too slow, the next step is a periodic snapshot of the book itself, so recovery loads that instead of rebuilding from the orders table.
