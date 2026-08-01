# Frontend Foundation

This document captures the first-step decisions for the trading platform frontend so future work stays aligned.

## Goals

- Keep the repository easy to reason about for both humans and coding agents.
- Leave room for realtime market data delivery over SSE or WebSockets.
- Establish a strict, modern TypeScript toolchain without pulling in unnecessary app complexity.
- Standardize on Node.js 24 for local development and CI parity.

## Stack choice

### Node.js 24

The project is pinned to Node 24 through the root package metadata and `.nvmrc`. That gives a clear runtime target for the frontend tooling and avoids drifting between developer machines.

### Vite

Vite gives fast local startup, simple production builds, and a shallow configuration surface. That makes it a good fit for incremental frontend work where we want the infrastructure to stay out of the way.

### React + TypeScript

React keeps the UI model familiar and flexible. TypeScript is enabled in strict mode so the app fails early when shapes drift, which matters in trading flows where data contracts tend to evolve quickly.

### ESLint + Prettier

ESLint handles code quality rules and Prettier handles formatting. The combination keeps reviews focused on behavior rather than style noise.

### Vitest + React Testing Library

This is the smallest practical test stack for a Vite app. It gives a fast feedback loop and keeps tests close to user-visible behavior.

## Realtime strategy

The frontend should not care whether a market feed comes from SSE or WebSockets. The next step will be to introduce a small transport layer with a shared message contract and adapter-style implementations.

That design keeps the rest of the app stable when the backend transport changes.

## Project structure

- `src/` contains the application code.
- `src/styles/` contains global styling and foundation tokens.
- `docs/` contains implementation notes and design decisions.

## Working rules

- Keep new features small and explicit.
- Prefer transport-agnostic interfaces near the domain boundary.
- Add a short document whenever a new cross-cutting decision is introduced.

## Current iteration status

This first iteration only establishes the foundation. It intentionally does not include trading flows, data fetching, or realtime transport code yet.