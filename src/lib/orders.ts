import { getBackendHost } from './api';

// Quantities and prices are decimal strings shown as-is; timestamps are RFC 3339.
export type Order = {
  id: string;
  pair_id: number;
  side: 'buy' | 'sell';
  quantity_ordered: string;
  quantity_filled: string;
  remaining_quantity: string;
  price_per_unit: string;
  status: string;
  created_at: string;
  last_updated_at: string | null;
};

type OrdersResponse = {
  orders: Order[];
};

const ordersPath = '/api/orders';

export async function fetchOrders(status: string, signal?: AbortSignal): Promise<Order[]> {
  const query = new URLSearchParams({ status });
  const response = await fetch(`${getBackendHost()}${ordersPath}?${query}`, {
    credentials: 'include',
    signal,
  });

  if (!response.ok) {
    throw new Error(`Orders request failed with status ${response.status}`);
  }

  const data = (await response.json()) as OrdersResponse;
  return data.orders;
}

export function formatOrderTime(timestamp: string) {
  return new Date(timestamp).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
