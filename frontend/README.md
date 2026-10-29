# Frontend

The web UI of the trading platform: a React single-page app that talks to the backend in the
repository root over its JSON API. The decision log lives in
[docs/frontend-foundation.md](docs/frontend-foundation.md).

## Pages

- **Login** (`/`): sign in, or sign up (which registers the user and then signs in). The
  backend sets the `access_token` cookie; every API request is sent with credentials.
- **Dashboard** (`/dashboard`): balances of the selected pair with a deposit dialog, open
  orders, a dialog to place a limit order, and the latest trades with the full, paginated
  trade history in a dialog. The *Portfolio value*, *24h P/L* and *Total trades* tiles are
  static placeholders.

Balances, open orders and trades are refreshed by polling (`src/lib/usePolling.ts`).

## Configuration

Copy [envs/.env.example](envs/.env.example) to `envs/.env` and fill in the values:

- `VITE_BACKEND_HOST`: base URL of the backend as the browser sees it, without a trailing
  slash; `http://localhost:8080` for the local setup.
- `VITE_BALANCE_POLL_INTERVAL_MS`, `VITE_ORDERS_POLL_INTERVAL_MS`,
  `VITE_TRADES_POLL_INTERVAL_MS`: how often the dashboard refreshes balances, open orders and
  the latest trades (default 5000, minimum 1000).

Vite inlines these values into the bundle at build time, so a change needs a rebuild (or a
restart of the dev server).

The backend accepts cross-origin requests only from `http://localhost:5173` by default
(`HTTP_FRONTEND_HOST`), so the UI must be served on that port: the dev server and the Docker
container both use it.

## Running

With Docker, from the repository root: `make up` builds this directory's image together with
the backend and serves the UI at http://localhost:5173. The image builds the app from
`envs/.env` and serves the static files with nginx (`Dockerfile`, `nginx.conf`); after
changing `envs/.env`, rebuild it with `make rebuild-up`.

For development, run the backend only (`docker compose up -d db app` in the root) and the Vite
dev server here. Requires Node.js 24 (`.nvmrc`).

- `npm install`
- `npm run dev`: dev server on http://localhost:5173
- `npm run build`: type-check and build into `dist/`
- `npm run lint`, `npm run typecheck`, `npm run format`
- `npm run test`: Vitest and React Testing Library

## Stack

- Vite for local development and production builds.
- React 19 with TypeScript (strict) and React Router.
- ESLint 9 flat config and Prettier for static checks and formatting.
- Vitest and React Testing Library for unit-level tests.
