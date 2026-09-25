interface ImportMetaEnv {
  readonly VITE_BACKEND_HOST?: string;
  readonly VITE_BALANCE_POLL_INTERVAL_MS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}