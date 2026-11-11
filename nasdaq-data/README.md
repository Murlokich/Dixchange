# Nasdaq ITCH data

Real Nasdaq order flow, used to drive the order book data structure benchmark with
realistic input. The source is the free Nasdaq TotalView-ITCH 5.0 sample day files
published at [emi.nasdaq.com/ITCH](https://emi.nasdaq.com/ITCH/Nasdaq%20ITCH/).

## Tool: itch-book

[itch-book](https://github.com/groovg/itch-book) (MIT) parses ITCH 5.0 day files,
rebuilds the order book of every symbol, and writes the results as Parquet tables
(messages, trades, BBO, halts, symbols, system events). It comes with a CLI
(`itch2parquet list | fetch | convert | verify | lobster`) and a Python API that returns
numpy / polars batches.

Why this one:
- Most ITCH parsers are built for streaming: they decode the live feed message by message
  for low-latency consumers. We don't need that. We work on static, recorded day files.
- What we need is a simple Python CLI and library that turns a whole day into tables we can
  analyse with polars, plus a book rebuild and a `verify` command to check the file
  (no unknown order references, no crossed books at the close).

## Setup

```bash
cd nasdaq-data
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
source .venv/bin/activate        # then itch2parquet is on the PATH
```

`.venv/` is gitignored; `requirements.txt` pins the exact versions.

## Dataset

`S061226-v50.txt.gz`: trading day 12 June 2026, 17.9 GB gzipped, about 1.3 billion
messages and 12,810 order books. It's the largest file on the server, i.e. the day with
the most order flow, so it gives the most data to choose symbols from.

Download it into `data/` (the download resumes if interrupted and checks the published md5):

```bash
itch2parquet fetch S061226-v50.txt.gz --dir data
```

Raw and converted data are not committed to git.

## ITCH events we use

Descriptions follow the official
[Nasdaq TotalView-ITCH 5.0 specification](https://www.nasdaqtrader.com/content/technicalsupport/specifications/dataproducts/NQTVITCHspecification.pdf);
the [itchfeed](https://github.com/bbalouki/itch) README has a short summary of every
message type.

**Session and symbol messages** (they don't change the book):

| Type | Message | Meaning |
|---|---|---|
| `S` | System Event | Market-wide (not per stock) marker of the day's phases: `O` start of messages, `S` start of system hours, `Q` start of market hours, `M` end of market hours, `E` end of system hours, `C` end of messages. Used to split the day into windows (pre-market, regular session, after hours) and as a check that the file is complete. |
| `R` | Stock Directory | Sent for every active symbol at the start of the day. Maps the stock locate code (used by all later messages, valid for that day only) to the symbol, and carries the market category, issue classification and ETP flags. |
| `H` | Stock Trading Action | Trading state of one stock across all U.S. markets: `H` halted, `P` paused (Nasdaq-listed only), `Q` quotation only, `T` trading. |
| `h` | Operational Halt | Interruption of service for one stock on one Nasdaq market center only; the stock can still trade on other markets. |

**Book messages** (they add, change or remove resting orders):

| Type | Message | Meaning |
|---|---|---|
| `A` | Add Order, no MPID attribution | A new resting order is added to the book (anonymous). |
| `F` | Add Order, MPID attribution | Same as `A`, but attributed to a market participant ID (an attributed order or quote). |
| `E` | Order Executed | A resting order is executed in whole or in part at its display price. |
| `C` | Order Executed With Price | A resting order is executed at a price different from its display price. Carries a printable flag: non-printable executions (e.g. in a cross) are counted later in a bulk print. |
| `X` | Order Cancel | Partial cancellation: some shares are removed from the order; it stays on the book under the same reference number. |
| `D` | Order Delete | The order is cancelled and all its remaining shares are removed from the book. |
| `U` | Order Replace | Atomic cancel + new order: the original order is removed and a new one is added with a new reference number, price and size. Side, symbol and MPID are inherited from the original order. |

Executions (`E`, `C`), cancels (`X`) and deletes (`D`) refer to the order's reference
number from its `A` / `F`. They are cumulative, and an order whose shares reach zero is
removed from the book.
