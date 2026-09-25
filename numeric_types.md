# Numeric Types

How this project represents money, quantities and prices, and why.

## Domain

### Pair, base, quote, price

- A **pair** `BASE/QUOTE` (e.g. `BTC/USDT`) is a market where one asset is traded for another.
- **Base** is the asset being bought or sold. **Quote** is the asset used to pay.
- **Price** is how much quote one *whole* unit of base costs. `BTC/USDT = 65000` means 1 BTC costs 65000 USDT.
- **Cost** (notional) of a trade is `quantity × price`, in the quote asset.

### Asset precision

- The smallest unit of an asset that the system can hold, written as the number of decimal places.
- All amounts of the asset are counted in this minimal unit:

| Asset | Minimal unit | Precision |
|---|---|---|
| USD | cent (0.01) | 2 |
| JPY | yen (1) | 0 |
| BTC | satoshi (0.00000001) | 8 |

- It is a property of the **asset**, and applies everywhere that asset appears: balances, reserves, order quantities.

### Lot size

- The smallest step by which an order's **quantity** can change. Every order must be a whole number of lots.
- It is a property of the **pair**. The same asset can have different lot sizes on different pairs.
- Why it exists:
  - Keeps orders a meaningful size (no orders for 0.00000001 BTC).
  - Keeps the order book compact and cheap to process.
  - Together with the tick, guarantees every trade cost can be paid exactly (see [the requirement](#requirement-for-parameters-setup-on-market-pair)).
- Example: 0.00001 BTC on `BTC/USDT`.

### Price tick

- The smallest step by which an order's **price** can change. Every price must be a whole number of ticks.
- It is a property of the **pair**.
- Why it exists:
  - Stops traders from jumping ahead in the queue by improving a price by a meaningless amount (e.g. 65000.0000001 vs 65000).
  - Fewer possible prices means fewer price levels in the order book.
  - Together with the lot, guarantees every trade cost can be paid exactly.
- It is chosen relative to the price level, typically ~0.01%–0.1% of the price:
  - Expensive asset (BTC ≈ 65000) → tick 0.01.
  - Cheap asset (≈ 0.00001) → tick 0.00000001.

### Pair price precision

- The number of decimal places a price on the pair can have, taken from the tick (tick 0.01 → price precision 2).
- A price is a ratio *quote per one whole base*, not an amount of either asset, so it cannot borrow either asset's precision. The pair defines it.

## Representation in code

Asset precision, lot size, price tick and price precision are domain parameters. They live in the database (`market.symbol`, `market.pair`) and every order, trade and balance depends on them. So the project needs a numeric representation that stores them, and all values derived from them, exactly.

### Options

- **Binary floating point (`float64`)** — rejected. It cannot represent most decimal fractions exactly (`0.1 + 0.2 != 0.3`), so errors accumulate in balances.
- **Exact decimal type** (`shopspring/decimal`, Postgres `NUMERIC`) — correct. A decimal is an integer plus a scale that the type tracks per value. Not chosen because:
  - Go decimal libraries are arbitrary-precision: every operation allocates on the heap, adding GC pressure and latency in the matching engine's hot path.
  - The scale travels with each value, so two values of the same field can silently have different scales.
- **Scaled integers (`int64` / `BIGINT`)** — chosen. Fixed-size, allocation-free, trivially comparable, and each field has one fixed scale.

### Decision

- The matching engine must use integers for performance.
- For consistency, the same representation is used everywhere else: the database, wallets and settlement. Every stored number is an integer, including lot size and price tick.
- Decimals appear only at the API boundary: parsing user input and formatting responses. Input that is not a whole number of lots or ticks is rejected, never rounded.
- The one risk integers add is **overflow** in intermediate products (`quantity × price`). Those multiplications must be done in 128-bit or checked arithmetic.

### Scaled integers

A stored integer `X` at scale `s` represents the real value `X · 10^−s`:

| Real value | Scale | Stored integer |
|---|---|---|
| 1.37 USD | 2 | `137` |
| 0.00123 BTC | 8 | `123000` |
| 65000.50 (price) | 2 | `6500050` |

The scale is not stored next to each value. It is owned by exactly one entity:

- **Asset precision** — scale of every amount of that asset (balances, reserves, order quantities, lot size).
- **Pair price precision** — scale of every price on that pair (order prices, price tick).

## Requirement for parameters setup on market pair

A market pair is valid only if:

```
(L · T) % 10^k == 0,     where k = b + p − q
```

| Symbol | Meaning |
|---|---|
| `L` | Lot size — smallest quantity step, as an integer at scale `b` |
| `T` | Price tick — smallest price step, as an integer at scale `p` |
| `b` | Precision of the **base** asset |
| `q` | Precision of the **quote** asset |
| `p` | Price precision of the **pair** |

If `k ≤ 0` the condition always holds.

**In words:** one lot at one tick must cost a whole number of the quote asset's minor units.

**Why it matters:** if the condition holds, the cost of *every* possible order is an exact integer in quote minor units, so settlement and fund reservation never need rounding. If it does not hold, some orders would cost a fraction of the smallest unit of money (e.g. half a cent) that cannot be debited from a wallet.

## Derivation

### Step 1 — Describe every possible order

Orders are only accepted in whole lots and whole ticks. So for some positive integers `n` (number of lots) and `m` (number of ticks):

```
Q = n · L      stored quantity, scale b
P = m · T      stored price,    scale p
```

### Step 2 — Turn stored integers into real values

```
real quantity = Q · 10^−b
real price    = P · 10^−p
```

### Step 3 — Compute the real cost

Cost is quantity times price:

```
real cost = Q · P · 10^−(b+p)
          = n · m · L · T · 10^−(b+p)
```

Multiplying two scaled numbers **adds their scales**: the product has up to `b + p` decimal places.

### Step 4 — Express the cost in quote minor units

The wallet stores the quote asset at scale `q`, so we multiply the real value by `10^q`:

```
cost in quote minor units = n · m · L · T · 10^−(b+p) · 10^q
                          = n · m · L · T / 10^(b+p−q)
                          = n · m · L · T / 10^k
```

`k = b + p − q` is the number of decimal places in the product beyond what the quote wallet can hold. Those extra digits must always be zero.

### Step 5 — Require an integer for every order

We need `n · m · L · T / 10^k` to be an integer for all `n, m ≥ 1`.

- **Necessary.** Take the smallest possible order, `n = m = 1`. Its cost is `L · T / 10^k`, so `L · T` must be divisible by `10^k`.
- **Sufficient.** If `L · T = c · 10^k` for some integer `c`, then any order costs `n · m · c` — a product of integers.

Therefore the condition is exactly `(L · T) % 10^k == 0`.

When `k ≤ 0`, dividing by `10^k` is multiplying by `10^−k`, which always yields an integer.

**Intuition:** every order is `n · m` copies of the smallest order (one lot at one tick). If the smallest order costs a whole number of minor units, every order does.

## Why precisions alone are not enough

`b` and `p` describe the *storage* scale — the maximum number of decimals a value could have. The lot and tick decide which values are *actually allowed*:

- A lot that ends in zeros (e.g. `L = 100` cents = 1 USD) means quantities never use those decimal places.
- Factors of the lot and tick can cancel out (e.g. `0.5 × 2 = 1`).

A rule built only from `b`, `p` and `q` cannot see either effect and would reject valid pairs. The divisibility check works with the actual lot and tick, so it accepts exactly the pairs that are safe.

## Examples

**USD/KCN** — buy dollars, pay in coins. `b = 2`, `q = 0`, `p = 0` → `k = 2`.
Lot 1 USD (`L = 100`), tick 1 KCN (`T = 1`):

```
L · T = 100,   100 % 10^2 = 0   ✓
3 lots at 5 ticks → 3 · 5 · 100 / 100 = 15 KCN   (3 USD at 5 KCN each)
```

**Cancelling factors.** Same assets, lot 0.5 USD (`L = 50`), tick 2 KCN (`T = 2`):

```
L · T = 100,   100 % 10^2 = 0   ✓
```

**Rejected.** Same assets, lot 0.01 USD (`L = 1`), tick 1 KCN (`T = 1`):

```
L · T = 1,     1 % 10^2 ≠ 0     ✗
```

One cent at 1 KCN per dollar would cost 0.01 KCN, which cannot exist.

**BTC/USDT.** `b = 8`, `q = 8`, `p = 2` → `k = 2`.
Lot 0.00001 BTC (`L = 1000`), tick 0.01 USDT (`T = 1`):

```
L · T = 1000,  1000 % 10^2 = 0  ✓
0.00123 BTC at 65000.50 → 123000 · 6500050 / 10^2 = 7995061500 = 79.95061500 USDT
```

**KCN/USD** — the inverse pair. `b = 0`, `q = 2`, `p = 2` → `k = 0`, always valid.

```
5 KCN at 0.37 USD → 5 · 37 / 10^0 = 185 cents = 1.85 USD
```

## Choosing parameters in practice

1. **Tick** (and with it `p`) — from the price level. A tick is typically around 0.01%–0.1% of the price: BTC at ~65000 uses 0.01, a coin at ~0.00001 uses 0.00000001.
2. **Lot** — from a sensible minimum order size.
3. **Validate** with the condition above. If it fails, increase the lot or coarsen the tick.
