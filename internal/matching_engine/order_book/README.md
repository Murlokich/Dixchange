# Order Book Documentation

## 1. What we need, from a financial standpoint
An order book is the in-memory structure that rapidly and correctly matches buy and sell orders for a single trading pair. Two rules make matching fair and predictable for clients:

* **Price priority** — the best price on the market always trades first. The best ask (lowest sell price) and the best bid (highest buy price) are matched before any worse-priced order.
* **Time priority (FIFO)** — among orders resting at the same price, the one that arrived first is filled first. Speed of arrival is a competitive advantage clients rely on, so it must be respected exactly.

Structurally this means resting orders naturally fall into two sides — an **ask side** and a **bid side** — each ordered by price (ascending for asks, descending for bids), and within a price, ordered by arrival time.

When a new order arrives, the engine looks at the opposite side: if the best resting price is beneficial to the incoming order, they trade, forming a fill. If the incoming order still has quantity left, it keeps matching against the next best resting order, looping until either its quantity is exhausted or it no longer matches the best remaining price. Any leftover quantity becomes a new resting order on its own side. This is what allows immediate trade registration on arrival, with partial fills as a first-class outcome rather than an edge case.

Everything below exists to make these two guarantees — price priority and time priority — cheap to enforce as order volume grows.

## 2. Interface
```
LoadMatchingEngine() *MatchingEngine // Creating matching engine based on orders in persistent storage
AddOrder(order Order) []Trade // Add new order and return matching trades
CancelOrder(id UUID) bool // Try to cancel order and reutrn the result
```

## 3. Operations we need
For the interface above to be fast, the underlying structure(s) must support:
* **Sorted traversal of price levels** — clients need to see the current market rate, i.e. the ordered set of active prices.
* **Quick access to, and extraction of, the best price level** — every incoming order is matched against the best ask/bid first, and that level may be fully consumed and need removing.
* **Quick insertion of a new resting order** — whatever quantity isn't matched must be added back cheaply.
* **FIFO ordering within a price level** — arrival order must be preserved so time priority holds.
* **Reverse lookup by order ID** — `CancelOrder` receives an ID, not a price, so the structure must support jumping straight to that order without scanning.

Cost of `AddOrder` in terms of these primitives:
`(best-price lookups × trades matched) + (extract-best-price × trades, only when a level empties out) + (one insert, if quantity remains after matching)`

## 4. Abstract data structures for the price levels
Two broad shapes of structure satisfy the operations in section 3:

### Option A — Ordered map keyed by price
An ordered map from price → FIFO queue of resting orders at that price. This satisfies the requirements directly:
* Keys are maintained in sorted order by construction, so sorted traversal (showing the market) and best-price access/extraction come for free from the map's own ordering — no separate sort step.
* Insert of a new resting order is either "append to an existing price's FIFO queue" or "insert a new key" — both operations the map is built for.
* The FIFO queue stored as the *value* at each price level gives time priority within that level for free.

What an ordered map does **not** give you is order-ID lookup — it's indexed by price, not by order ID. `CancelOrder(id)` would otherwise mean scanning every price level's queue, which is unacceptable. This is why a **reverse lookup DS** (a plain hash map from order ID → the specific price key and the exact node inside that price's FIFO queue) is required alongside the ordered map: it turns cancellation into a direct jump to the node, independent of how many price levels or orders exist elsewhere.

### Option B — Array over a predefined price range
If the tick size and price bounds of a market are known ahead of time (e.g. ticks of $0.01 between $0 and $1000), every possible price level can be assigned a fixed array index: `index = (price - minPrice) / tickSize`. Reaching any price level is then pure arithmetic plus array indexing — O(1) — with the best possible cache locality, since neighboring price levels sit in contiguous memory. The cost is that it requires the range and tick size to be pinned down in advance, and it commits memory (or needs resizing logic) for the whole range regardless of how sparsely it's populated.

### Decision
This project uses **Option A (ordered map)**. This is an artificial exchange without an established price range or tick size yet, so there is nothing to fix an array's bounds against. Option B is a legitimate future optimization once a real production system has settled on real market parameters — it's noted here as a deliberate improvement to revisit, not a rejected idea.

## 5. Choosing the ordered map's data structure
An "ordered map" is an interface, not an implementation — it still needs a concrete self-balancing structure underneath. `n` = number of resting orders, `P` = number of distinct active price levels (`P ≤ n`), `K` = requested depth for a Top K query. Here, Insert/Delete/Top operate on **price-level keys**, not individual orders (order-level insert/remove within a level's FIFO queue is O(1) regardless of which structure is chosen).

| Data structure | Top K | Top 1 | Extract Top | Insert | Arbitrary Delete | Summary |
|---|---|---|---|---|---|---|
| Skip List | O(K) | O(1) | O(log P) | O(log P) | O(log P) | Probabilistically balanced, simple ordered traversal. |
| Red-Black Tree | O(K + log P) | O(1)* | O(log P) | O(log P) | O(log P) | Deterministic worst-case O(log P) on every op. |
| AVL Tree | O(K + log P) | O(1)* | O(log P) | O(log P) | O(log P) | Same asymptotics as Red-Black, stricter balancing. |
| B-Tree | O(K + log P) | O(log P) | O(log P) | O(log P) | O(log P) | High branching factor, cache/page friendly. |

\* assumes a cached pointer to the min/max node, updated on insert/delete.

**Red-Black Tree** is the canonical, "textbook optimal" balanced BST — it's what backs `std::map` in C++ and `TreeMap` in Java, and its O(log P) bounds are as good as this family gets. Its weakness is cache locality: each node is a separate heap allocation linked by pointers, so walking or rebalancing the tree means chasing pointers scattered across memory, which is slow on modern CPUs regardless of the Big-O.

**AVL Tree** is close to Red-Black in every other respect, but keeps a stricter balance invariant, which means more rotations — and therefore more writes — on insert and delete. Read-heavy, write-light workloads is where it distinguishes itself from Red-Black; here, both sides see frequent inserts and deletes, so that advantage doesn't apply.

**Skip List** is the easiest of the four to implement correctly — no rotations, just randomized layering — at the cost of no deterministic worst case: its O(log P) is a probabilistic guarantee, and an unlucky sequence of inserts can degrade it.

**B-Tree** is the cache-locality-oriented answer to the same problem: instead of one key per node, each node holds many keys in a contiguous block, so a lookup touches far fewer cache lines / memory pages than a pointer-chasing BST for the same P. It's the structure Rust's standard library chose for its main ordered map (`BTreeMap`) for exactly this reason.

## 6. Final decision
This order book uses a **B-Tree** for its ordered price-level maps.

* **Go ecosystem fit** — of the four options, B-Tree is the only one with a mature, ready-to-use implementation available, `google/btree`, maintained by Google and about as close to a de facto standard as the Go ecosystem has for ordered containers; Red-Black Tree, AVL Tree, and Skip List have no equivalently established Go package and would mean implementing and maintaining a balanced tree by hand.
* **Cache locality** — as noted in section 5, its contiguous, multi-key nodes give it a real-world performance edge over Red-Black/AVL/Skip List that the shared O(log P) bounds don't capture.
* **Overall trend** — the same reasoning is why modern systems languages moved this direction for in-memory ordered maps (e.g. Rust's `BTreeMap`); this decision follows that trend rather than the classic C++/Java default of a Red-Black tree.

The array-over-a-fixed-price-range design from section 4 remains the natural next step once this exchange has real, bounded price/tick parameters to build it against.
