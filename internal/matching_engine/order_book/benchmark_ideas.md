# Order book data structures: implementation ideas

Working notes for benchmarking the price-level structure of the order book. Ideas to be
implemented, benchmarked and compared, not decisions.

## Current state

- `sides.go`: `PriceLevel`, `less` and the `bookSide` interface (code that doesn't depend on
  the data structure).
- `btree.go`: `asks` / `bids` wrapping `google/btree.BTreeG[PriceLevel]`; best ask = `Min`,
  best bid = `Max`.
- `BTreeG` compares through the `less` func value: an indirect, non-inlinable call on every
  comparison. Keep it as the library baseline, and note this cost when comparing it with
  the structures written here.

## Common to all ideas

- Key the structures by price (`int64`) with the order queue (`*list.List`) as the value,
  instead of storing `PriceLevel` with a `less` func. Comparisons become native `<` on
  `int64`.
- Every structure provides the same operations: `Len`, `Get`, `Put`, `Delete`, `Min`, `Max`.

## Idea 1: K, V as generics

- Each structure is a generic ordered map `Map[K cmp.Ordered, V any]`, independent of the
  order book, instantiated as `Map[int64, *list.List]`.
- Why it should be cheap: Go compiles generics per GC shape. `int64` is a value shape, so it
  gets its own compiled copy and `<` stays native. `*list.List` falls into the shared pointer
  shape, but the structure only stores and returns it, never calls methods on it.
- Possible costs: the hidden dictionary argument, and the compiler being more conservative
  about inlining generic code.

## Idea 2: direct implementation

- The same structures with no type parameters, written for `int64` keys and `*list.List`
  values only. The code is idea 1 with `K` and `V` substituted.
- Note: not fully generic-free either, since `slices.BinarySearch` / `Insert` / `Delete`
  are generic.
- Benchmark one structure (e.g. sorted array) in both versions. If they're within noise,
  use idea 1 for all structures and cite the check; if idea 2 clearly wins, write them all
  directly.

## Idea 3: side separation through an interface

How asks and bids pick their best price on top of a structure. The thing to measure is the
number of indirect calls per operation.

- **3a. Concrete side struct holding an `orderedMap` interface + `bestIsMax` flag.**
  No `bookSide` interface: `Book` holds `*bookSide`, `getBestPriceLevel` branches on the flag
  (fixed per side, so always predicted). One indirect call per operation (into the
  structure). Least code: a structure only implements `orderedMap`, no adapters. Loses the
  per-side layout choice (e.g. a sorted array wants the best price at the cheap end); that
  choice moves into the structure's constructor.
- **3b. Per-structure adapter pairs embedding the concrete type** (`btreeAsks`,
  `btreeBids`, `sortedArrayAsks`, ...) behind the `bookSide` interface. One indirect call
  (`bookSide`), the structure's methods can inline. About 10 lines of boilerplate per
  structure; each pair can choose its own layout per side.
- **3c. `asks` / `bids` structs embedding the `orderedMap` interface behind the `bookSide`
  interface.** Two indirect calls per operation (`bookSide`, then `orderedMap`; promoted
  methods are wrappers that dispatch again). Estimated ~1-2 ns per operation, constant across
  structures, but it shrinks relative gaps in microbenchmarks of cheap operations.

## Variants to measure

Choices made for the first version of each structure in `ds/`, with the alternative to
benchmark against.

- **Skip list, tower allocation.** Now: the tower is a separate slice, so 2 allocations per
  node. Variant: a fixed `[maxLevel]*node` array inside the node, so 1 allocation but
  `maxLevel` pointers on every node.
- **Skip list, order of the bid side.** Now: the bid side is built descending (`asc = false`),
  so the best price sits at the cheap head end. Variant: ascending + tail pointer for both
  sides, to measure how much the head/tail asymmetry costs.
- **B-tree, search inside a node.** Now: linear scan (as Rust's `BTreeMap`; branch-predictable,
  streams through the keys). Variant: binary search (as `google/btree`). Expected to interact
  with the degree: linear should win for small nodes, binary for large ones.
- **B-tree, degree.** Sweep the minimum degree t = 2, 4, 8, 16, 32, 64. t = 2 is a 2-3-4 tree,
  equivalent to a red-black tree.

## How to measure

- `go test -bench -benchmem -count=N` + `benchstat` for time and allocs per op.
- `go build -gcflags=-m` to see what inlines and what escapes.
- Per-node heap objects (trees, skip lists) vs contiguous memory (sorted array) are expected
  to matter more than any of the dispatch questions above.
