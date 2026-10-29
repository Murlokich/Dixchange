import { formatOrderTime } from '../lib/orders';
import type { Trade } from '../lib/trades';

type TradesTableProps = {
  trades: Trade[];
  pairLabels: Map<number, string>;
};

export default function TradesTable({ trades, pairLabels }: TradesTableProps) {
  return (
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
        {trades.map((trade) => (
          <tr key={trade.id}>
            <td>{pairLabels.get(trade.pair_id) ?? `Pair #${trade.pair_id}`}</td>
            <td className={trade.side === 'buy' ? 'side-buy' : 'side-sell'}>{trade.side}</td>
            <td>{trade.amount}</td>
            <td>{trade.price}</td>
            <td>{trade.total}</td>
            <td>{formatOrderTime(trade.executed_at)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
