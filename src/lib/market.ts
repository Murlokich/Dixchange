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

// "main" is the available amount; both values are decimal strings shown as-is.
export type SymbolBalance = {
  main: string;
  reserved: string;
};

export type PairBalance = {
  pair_id: number;
  base: SymbolBalance;
  quote: SymbolBalance;
};

const pairsPath = '/api/market/pairs';
const balancePath = (pairId: number) => `/api/balance/${pairId}`;
const depositPath = '/api/balance/deposit';

const defaultBalancePollIntervalMs = 5000;
const minBalancePollIntervalMs = 1000;

let pairsRequest: Promise<MarketPair[]> | null = null;

function getBackendHost() {
  const backendHost = import.meta.env.VITE_BACKEND_HOST?.trim();

  if (!backendHost) {
    throw new Error('Set VITE_BACKEND_HOST in envs/.env before continuing.');
  }

  return backendHost;
}

// Falls back to the default when unset or invalid, and never polls faster than the minimum.
export function getBalancePollIntervalMs() {
  const configured = Number(import.meta.env.VITE_BALANCE_POLL_INTERVAL_MS);

  if (!Number.isFinite(configured) || configured <= 0) {
    return defaultBalancePollIntervalMs;
  }

  return Math.max(configured, minBalancePollIntervalMs);
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

export async function fetchPairBalance(pairId: number, signal?: AbortSignal): Promise<PairBalance> {
  const response = await fetch(`${getBackendHost()}${balancePath(pairId)}`, {
    credentials: 'include',
    signal,
  });

  if (!response.ok) {
    throw new Error(`Balance request failed with status ${response.status}`);
  }

  return (await response.json()) as PairBalance;
}

export async function depositBalance(symbolId: number, amount: string): Promise<void> {
  const response = await fetch(`${getBackendHost()}${depositPath}`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ amount, symbol_id: symbolId }),
  });

  if (!response.ok) {
    throw new Error(`Deposit request failed with status ${response.status}`);
  }
}
