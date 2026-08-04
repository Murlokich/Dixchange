import { useState } from 'react';
import { getStoredEmail } from '../lib/session';

type Balance = {
  currency: string;
  amount: number;
};

type OpenOrder = {
  id: string;
  pair: string;
  side: 'buy' | 'sell';
  type: string;
  amount: number;
  price: number;
  status: string;
};

type Trade = {
  id: string;
  pair: string;
  side: 'buy' | 'sell';
  amount: number;
  price: number;
  total: number;
  executedAt: string;
};

type Stat = {
  label: string;
  value: string;
  tone?: 'positive' | 'negative';
};

const balances: Balance[] = [
  { currency: 'USD', amount: 12480.32 },
  { currency: 'KCN', amount: 3250.5 },
];

const openOrders: OpenOrder[] = [
  { id: 'ORD-1042', pair: 'KCN/USD', side: 'buy', type: 'Limit', amount: 150, price: 4.82, status: 'Open' },
  { id: 'ORD-1041', pair: 'KCN/USD', side: 'sell', type: 'Limit', amount: 80, price: 5.1, status: 'Open' },
  {
    id: 'ORD-1038',
    pair: 'KCN/USD',
    side: 'buy',
    type: 'Market',
    amount: 25,
    price: 4.95,
    status: 'Partially filled',
  },
];

const tradeHistory: Trade[] = [
  { id: 'TRD-9231', pair: 'KCN/USD', side: 'buy', amount: 200, price: 4.7, total: 940, executedAt: '2026-08-03 14:22' },
  { id: 'TRD-9214', pair: 'KCN/USD', side: 'sell', amount: 60, price: 5.05, total: 303, executedAt: '2026-08-02 09:47' },
  {
    id: 'TRD-9188',
    pair: 'KCN/USD',
    side: 'buy',
    amount: 400,
    price: 4.55,
    total: 1820,
    executedAt: '2026-07-30 18:03',
  },
];

const stats: Stat[] = [
  { label: 'Portfolio value', value: '$18,640.90' },
  { label: '24h P/L', value: '+2.4%', tone: 'positive' },
  { label: 'Open orders', value: String(openOrders.length) },
  { label: 'Total trades', value: '128' },
];

const orderMenuOptions = ['Market order', 'Limit order'];

function formatAmount(amount: number) {
  return amount.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
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
  const [isOrderMenuOpen, setIsOrderMenuOpen] = useState(false);

  return (
    <main className="dashboard-page">
      <header className="dashboard-topbar">
        <div className="brand">
          <span className="brand-mark">
            <BrandMark />
          </span>
          <span>Trading Platform</span>
        </div>

        <div className="topbar-right">
          <div className="balance-panel">
            {balances.map((balance) => (
              <div className="balance-item" key={balance.currency}>
                <span className="balance-label">{balance.currency}</span>
                <span className="balance-value">{formatAmount(balance.amount)}</span>
              </div>
            ))}
            <button type="button" className="deposit-button">
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
          {openOrders.length === 0 ? (
            <p className="empty-state">No open orders yet.</p>
          ) : (
            <table className="orders-table">
              <thead>
                <tr>
                  <th>Pair</th>
                  <th>Side</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Price</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {openOrders.map((order) => (
                  <tr key={order.id}>
                    <td>{order.pair}</td>
                    <td className={order.side === 'buy' ? 'side-buy' : 'side-sell'}>{order.side}</td>
                    <td>{order.type}</td>
                    <td>{formatAmount(order.amount)}</td>
                    <td>{formatAmount(order.price)}</td>
                    <td>{order.status}</td>
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
            onClick={() => setIsOrderMenuOpen((open) => !open)}
            aria-expanded={isOrderMenuOpen}
          >
            <PlusIcon />
            Create order
          </button>

          {isOrderMenuOpen ? (
            <div className="create-order-menu" role="menu">
              {orderMenuOptions.map((option) => (
                <button type="button" key={option} role="menuitem" onClick={() => setIsOrderMenuOpen(false)}>
                  {option}
                </button>
              ))}
            </div>
          ) : null}
        </section>

        <section className="panel panel-history" aria-labelledby="trade-history-title">
          <h2 id="trade-history-title">Trade history</h2>
          {tradeHistory.length === 0 ? (
            <p className="empty-state">No trades yet.</p>
          ) : (
            <table className="history-table">
              <thead>
                <tr>
                  <th>Pair</th>
                  <th>Side</th>
                  <th>Amount</th>
                  <th>Price</th>
                  <th>Total</th>
                  <th>Executed</th>
                </tr>
              </thead>
              <tbody>
                {tradeHistory.map((trade) => (
                  <tr key={trade.id}>
                    <td>{trade.pair}</td>
                    <td className={trade.side === 'buy' ? 'side-buy' : 'side-sell'}>{trade.side}</td>
                    <td>{formatAmount(trade.amount)}</td>
                    <td>{formatAmount(trade.price)}</td>
                    <td>{formatAmount(trade.total)}</td>
                    <td>{trade.executedAt}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </main>
  );
}
