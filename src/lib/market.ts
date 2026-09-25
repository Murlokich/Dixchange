export type MarketSymbol = {
  id: number;
  code: string;
  precision: number;
};

// Decimal fields stay as strings exactly as the backend sends them; they are never parsed into numbers.
export type MarketPair = {
  id: number;
  // Ordered [base, quote], e.g. USD/KCN: lot_size is in base units, prices are quote per one base.
  symbols: MarketSymbol[];
  lot_size: string;
  price_tick: string;
  price_precision: number;
};

type PairsResponse = {
  pairs: MarketPair[];
};

type BalanceResponse = {
  amount: number;
};

const pairsPath = '/api/market/pairs';
const balancePath = (symbolId: number) => `/api/balances/${symbolId}`;

let pairsRequest: Promise<MarketPair[]> | null = null;

function getBackendHost() {
  const backendHost = import.meta.env.VITE_BACKEND_HOST?.trim();

  if (!backendHost) {
    throw new Error('Set VITE_BACKEND_HOST in envs/.env before continuing.');
  }

  return backendHost;
}

export function formatPairLabel(pair: MarketPair) {
  return pair.symbols.map((symbol) => symbol.code).join('/');
}

// Pairs rarely change, so the request is shared and made only once per page load.
export function fetchPairs(): Promise<MarketPair[]> {
  pairsRequest ??= (async () => {
    const response = await fetch(`${getBackendHost()}${pairsPath}`, { credentials: 'include' });

    if (!response.ok) {
      throw new Error(`Pairs request failed with status ${response.status}`);
    }

    const data = (await response.json()) as PairsResponse;
    return data.pairs;
  })().catch((error: unknown) => {
    pairsRequest = null;
    throw error;
  });

  return pairsRequest;
}

export async function fetchBalance(symbolId: number, signal?: AbortSignal): Promise<number> {
  const response = await fetch(`${getBackendHost()}${balancePath(symbolId)}`, {
    credentials: 'include',
    signal,
  });

  if (!response.ok) {
    throw new Error(`Balance request failed with status ${response.status}`);
  }

  const data = (await response.json()) as BalanceResponse;
  return data.amount;
}
