import { useEffect, useMemo, useRef, useState } from 'react';
import {
  fetchPairBalance,
  fetchPairs,
  formatPairLabel,
  type MarketPair,
  type PairBalance,
} from '../lib/market';
import { fetchOrders, formatOrderTime, type Order } from '../lib/orders';
import CancelOrderButton from '../components/CancelOrderButton';
import CreateOrderDialog from '../components/CreateOrderDialog';
import DepositDialog from '../components/DepositDialog';
import TradeHistoryDialog from '../components/TradeHistoryDialog';
import TradesTable from '../components/TradesTable';
import { getStoredEmail } from '../lib/session';
import { fetchTrades, type Trade } from '../lib/trades';
import { readPollIntervalMs, usePolling, type PollFetcher } from '../lib/usePolling';

type Stat = {
  label: string;
  value: string;
  tone?: 'positive' | 'negative';
};

const balancePollIntervalMs = readPollIntervalMs(import.meta.env.VITE_BALANCE_POLL_INTERVAL_MS);
const ordersPollIntervalMs = readPollIntervalMs(import.meta.env.VITE_ORDERS_POLL_INTERVAL_MS);
const tradesPollIntervalMs = readPollIntervalMs(import.meta.env.VITE_TRADES_POLL_INTERVAL_MS);

const openOrderStatuses = ['open', 'partially_filled'];

const fetchOpenOrders: PollFetcher<Order[]> = (signal) => fetchOrders(openOrderStatuses, signal);
// The dashboard widget shows only the newest page; older trades live in the trade history dialog.
const fetchLatestTrades: PollFetcher<Trade[]> = async (signal) =>
  (await fetchTrades(null, signal)).items;

function ChevronIcon() {
  return (
    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
      <path
        d="M6 9L12 15L18 9"
        stroke="currentColor"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function BrandMark() {
  return (
    <svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
      <path
        d="M4 17L10 10L14 14L20 6"
        stroke="currentColor"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" aria-hidden="true">
      <path d="M12 5V19M5 12H19" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" />
    </svg>
  );
}

export default function DashboardPage() {
  const email = getStoredEmail() ?? 'Unknown user';
  const initial = email.charAt(0).toUpperCase() || '?';
  const [isCreateOrderOpen, setIsCreateOrderOpen] = useState(false);
  const [isTradeHistoryOpen, setIsTradeHistoryOpen] = useState(false);
  const [pairs, setPairs] = useState<MarketPair[]>([]);
  const [selectedPairId, setSelectedPairId] = useState<number | null>(null);
  const [isPairMenuOpen, setIsPairMenuOpen] = useState(false);
  const [isDepositOpen, setIsDepositOpen] = useState(false);
  const pairSelectorRef = useRef<HTMLDivElement>(null);

  const selectedPair = pairs.find((pair) => pair.id === selectedPairId) ?? null;

  const fetchSelectedPairBalance = useMemo<PollFetcher<PairBalance> | null>(() => {
    if (selectedPairId === null) {
      return null;
    }
    return async (signal) => {
      const balance = await fetchPairBalance(selectedPairId, signal);
      // The response names its pair, so a late answer for a previously selected pair is dropped.
      return balance.pair_id === selectedPairId ? balance : null;
    };
  }, [selectedPairId]);

  const { data: pairBalance, refresh: refreshBalance } = usePolling(
    fetchSelectedPairBalance,
    balancePollIntervalMs,
  );
  const { data: openOrders, refresh: refreshOrders } = usePolling(
    fetchOpenOrders,
    ordersPollIntervalMs,
  );
  const { data: latestTrades, refresh: refreshTrades } = usePolling(
    fetchLatestTrades,
    tradesPollIntervalMs,
  );

  const pairLabels = new Map(pairs.map((pair) => [pair.id, formatPairLabel(pair)]));
  const stats: Stat[] = [
    { label: 'Portfolio value', value: '$18,640.90' },
    { label: '24h P/L', value: '+2.4%', tone: 'positive' },
    { label: 'Open orders', value: openOrders ? String(openOrders.length) : '—' },
    { label: 'Total trades', value: '128' },
  ];
  const [baseSymbol, quoteSymbol] = selectedPair?.symbols ?? [];
  const balances = [
    { symbol: baseSymbol, balance: pairBalance?.base },
    { symbol: quoteSymbol, balance: pairBalance?.quote },
  ];

  useEffect(() => {
    let isActive = true;

    fetchPairs()
      .then((loadedPairs) => {
        if (!isActive) {
          return;
        }
        setPairs(loadedPairs);
        setSelectedPairId((current) => current ?? loadedPairs[0]?.id ?? null);
      })
      .catch((error: unknown) => console.error(error));

    return () => {
      isActive = false;
    };
  }, []);

  useEffect(() => {
    if (!isPairMenuOpen) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (!pairSelectorRef.current?.contains(event.target as Node)) {
        setIsPairMenuOpen(false);
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, [isPairMenuOpen]);

  return (
    <main className="dashboard-page">
      <header className="dashboard-topbar">
        <div className="brand">
          <span className="brand-mark">
            <BrandMark />
          </span>
          <span>Trading Platform</span>
        </div>

        <div className="pair-selector" ref={pairSelectorRef}>
          <button
            type="button"
            className="pair-trigger"
            onClick={() => setIsPairMenuOpen((open) => !open)}
            aria-haspopup="menu"
            aria-expanded={isPairMenuOpen}
            disabled={pairs.length === 0}
            data-pair-id={selectedPair?.id}
          >
            <span className="pair-label">
              {selectedPair ? formatPairLabel(selectedPair) : 'Loading…'}
            </span>
            {selectedPair ? <span className="pair-id">#{selectedPair.id}</span> : null}
            <ChevronIcon />
          </button>

          {isPairMenuOpen ? (
            <div className="pair-menu" role="menu">
              {pairs.map((pair) => (
                <button
                  type="button"
                  key={pair.id}
                  role="menuitemradio"
                  aria-checked={pair.id === selectedPairId}
                  onClick={() => {
                    setSelectedPairId(pair.id);
                    setIsPairMenuOpen(false);
                  }}
                >
                  <span>{formatPairLabel(pair)}</span>
                  <span className="pair-id">#{pair.id}</span>
                </button>
              ))}
            </div>
          ) : null}
        </div>

        <div className="topbar-right">
          <div className="balance-panel">
            {balances.map(({ symbol, balance }) =>
              symbol ? (
                <div className="balance-item" key={symbol.id}>
                  <span className="balance-label">{symbol.code}</span>
                  <span className="balance-value">{balance?.main ?? '—'}</span>
                  <span className="balance-reserved">{balance?.reserved ?? '—'} reserved</span>
                </div>
              ) : null,
            )}
            <button
              type="button"
              className="deposit-button"
              onClick={() => setIsDepositOpen(true)}
              disabled={!selectedPair}
            >
              <PlusIcon />
              Deposit
            </button>
          </div>

          <div className="account-chip">
            <span className="account-avatar" aria-hidden="true">
              {initial}
            </span>
            <span className="account-email">{email}</span>
          </div>
        </div>
      </header>

      <section className="stats-section" aria-label="Account statistics">
        {stats.map((stat) => (
          <div className="stat-tile" key={stat.label}>
            <span className="stat-label">{stat.label}</span>
            <span className={`stat-value ${stat.tone ?? ''}`}>{stat.value}</span>
          </div>
        ))}
      </section>

      <div className="dashboard-grid">
        <section className="panel panel-orders" aria-labelledby="open-orders-title">
          <h2 id="open-orders-title">Open orders</h2>
          {openOrders === null ? (
            <p className="empty-state">Loading open orders…</p>
          ) : openOrders.length === 0 ? (
            <p className="empty-state">No open orders yet.</p>
          ) : (
            <table className="orders-table">
              <thead>
                <tr>
                  <th>Pair</th>
                  <th>Side</th>
                  <th>Price</th>
                  <th>Remaining / Ordered</th>
                  <th>Created</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {openOrders.map((order) => (
                  <tr key={order.id}>
                    <td>{pairLabels.get(order.pair_id) ?? `Pair #${order.pair_id}`}</td>
                    <td className={order.side === 'buy' ? 'side-buy' : 'side-sell'}>{order.side}</td>
                    <td>{order.price_per_unit}</td>
                    <td>
                      {order.remaining_quantity} / {order.quantity_ordered}
                    </td>
                    <td
                      title={
                        order.last_updated_at
                          ? `Last updated ${formatOrderTime(order.last_updated_at)}`
                          : 'Not updated since creation'
                      }
                    >
                      {formatOrderTime(order.created_at)}
                    </td>
                    <td className="order-actions">
                      <CancelOrderButton
                        order={order}
                        onCancelled={() => {
                          // Cancelling releases reserved funds, so the balance changes too.
                          refreshOrders();
                          refreshBalance();
                        }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        <section className="panel panel-create create-order-panel" aria-labelledby="create-order-title">
          <h2 id="create-order-title">Create order</h2>
          <button
            type="button"
            className="primary-action create-order-trigger"
            onClick={() => setIsCreateOrderOpen(true)}
            disabled={pairs.length === 0}
          >
            <PlusIcon />
            Create order
          </button>
        </section>

        <section className="panel panel-history" aria-labelledby="trade-history-title">
          <div className="panel-header">
            <h2 id="trade-history-title">Trade history</h2>
            <button type="button" className="panel-link" onClick={() => setIsTradeHistoryOpen(true)}>
              View all
            </button>
          </div>
          {latestTrades === null ? (
            <p className="empty-state">Loading trades…</p>
          ) : latestTrades.length === 0 ? (
            <p className="empty-state">No trades yet.</p>
          ) : (
            <TradesTable trades={latestTrades} pairLabels={pairLabels} />
          )}
        </section>
      </div>

      {isDepositOpen && selectedPair ? (
        <DepositDialog
          symbols={selectedPair.symbols}
          onClose={() => setIsDepositOpen(false)}
          onDeposited={refreshBalance}
        />
      ) : null}

      {isTradeHistoryOpen ? (
        <TradeHistoryDialog pairLabels={pairLabels} onClose={() => setIsTradeHistoryOpen(false)} />
      ) : null}

      {isCreateOrderOpen ? (
        <CreateOrderDialog
          pairs={pairs}
          initialPairId={selectedPairId}
          onClose={() => setIsCreateOrderOpen(false)}
          onCreated={() => {
            // A new order reserves funds and may match immediately, so all three can change.
            refreshOrders();
            refreshBalance();
            refreshTrades();
          }}
        />
      ) : null}
    </main>
  );
}
