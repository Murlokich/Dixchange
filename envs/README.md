# Environment files

- `app/`: configuration of the application, one file per environment.
  - `.env`: local development and docker-compose. Not committed: it holds secrets (JWT keys).
    Start from `.env.example`.
  - `.env.example`: template of `.env`, without secrets.
  - `.env.github_runner`: replaces `.env` in the e2e job of the GitHub workflow; no secrets.
  - `e2e.env`: the few app settings the e2e tests deliberately run differently (e.g. cheaper
    password hashing). Loaded on top of `.env` by `make test-e2e`.
- `tests/`: settings of the tests themselves, which the application never reads.
  - `e2e.env`: settings of the e2e suites (e.g. the size of the randomized suite).

`make test-e2e` loads `tests/e2e.env`, `app/e2e.env` and `app/.env`; the first file that sets
a variable wins.
