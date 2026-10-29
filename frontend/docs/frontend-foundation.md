# Frontend Foundation

This document captures the first-step decisions for the trading platform frontend so future work stays aligned.

## Goals

- Keep the frontend easy to reason about for both humans and coding agents.
- Establish a strict, modern TypeScript toolchain without pulling in unnecessary app complexity.
- Standardize on Node.js 24 for local development and CI parity.

## Stack choice

### Node.js 24

The project is pinned to Node 24 through `engines` in `package.json` and `.nvmrc`. That gives a clear runtime target for the frontend tooling and avoids drifting between developer machines.

### Vite

Vite gives fast local startup, simple production builds, and a shallow configuration surface. That makes it a good fit for incremental frontend work where we want the infrastructure to stay out of the way.

### React + TypeScript

React keeps the UI model familiar and flexible. TypeScript is enabled in strict mode so the app fails early when shapes drift, which matters in trading flows where data contracts tend to evolve quickly.

### ESLint + Prettier

ESLint handles code quality rules and Prettier handles formatting. The combination keeps reviews focused on behavior rather than style noise.

### Vitest + React Testing Library

This is the smallest practical test stack for a Vite app. It gives a fast feedback loop and keeps tests close to user-visible behavior.

## Data refresh

The backend has no push channel, so the dashboard polls: `usePolling` (`src/lib/usePolling.ts`) refetches balances, open orders and the latest trades on an interval configured per data set in `envs/.env`.

## Project structure

- `src/pages/` contains the routed pages (login, dashboard).
- `src/components/` contains the dialogs and tables the pages use.
- `src/lib/` contains the API clients, polling and input helpers.
- `src/styles/` contains global styling and foundation tokens.
- `docs/` contains implementation notes and design decisions.
- `Dockerfile` and `nginx.conf` build the static bundle and serve it in docker compose.

## Working rules

- Keep new features small and explicit.
- Prefer transport-agnostic interfaces near the domain boundary.
- Add a short document whenever a new cross-cutting decision is introduced.