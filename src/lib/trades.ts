import { getBackendHost } from './api';
import type { OrderSide } from './orders';
import type { CursorPage } from './useCursorPages';

// Amount, price and total are decimal strings shown as-is; executed_at is RFC 3339.
export type Trade = {
  id: string;
  pair_id: number;
  order_id: string;
  side: OrderSide;
  amount: string;
  price: string;
  total: string;
  executed_at: string;
};

type TradesResponse = {
  trades: Trade[];
  next_cursor: string | null;
};

const tradesPath = '/api/trades';

// Returns the newest trades first; pass the previous page's cursor to get the next (older) page.
export async function fetchTrades(
  cursor: string | null,
  signal?: AbortSignal,
): Promise<CursorPage<Trade>> {
  const query = cursor === null ? '' : `?${new URLSearchParams({ cursor })}`;
  const response = await fetch(`${getBackendHost()}${tradesPath}${query}`, {
    credentials: 'include',
    signal,
  });

  if (!response.ok) {
    throw new Error(`Trades request failed with status ${response.status}`);
  }

  const data = (await response.json()) as TradesResponse;
  return { items: data.trades, nextCursor: data.next_cursor };
}
