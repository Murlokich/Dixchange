interface ImportMetaEnv {
  readonly VITE_BACKEND_HOST?: string;
  readonly VITE_BALANCE_POLL_INTERVAL_MS?: string;
  readonly VITE_ORDERS_POLL_INTERVAL_MS?: string;
  readonly VITE_TRADES_POLL_INTERVAL_MS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}