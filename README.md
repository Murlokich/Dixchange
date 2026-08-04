# Frontend

This repo now starts as a small, documented frontend foundation for a trading platform.

The decision log lives in [docs/frontend-foundation.md](docs/frontend-foundation.md).

## Runtime

- Node.js 24

## Backend host

- Set the backend host in [envs/.env](envs/.env).
- The app reads `VITE_BACKEND_HOST` from that file and uses it for the sign-up POST request.

## Current screen

- Login page design with email and password fields.
- Primary sign in action and secondary sign up action.
- Sign up now posts to `POST /api/auth/register` on the configured backend host.

## Stack

- Vite for fast local development and production builds.
- React 19 with TypeScript for the UI layer.
- ESLint 9 flat config and Prettier for formatting and static checks.
- Vitest and React Testing Library for unit-level coverage.

## Why this shape

- The codebase stays small and explicit, which makes it easier for people and agents to reason about.
- SSE and WebSocket integrations will live behind a thin realtime client layer so the rest of the app does not care which transport is used.
- Path aliases and a strict TypeScript setup keep imports predictable as the app grows.

## Commands

- `npm install`
- `npm run dev`
- `npm run build`
- `npm run lint`
- `npm run test`

## Next step

The next iteration should wire the login form to your backend contract.