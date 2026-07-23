# Monolith

## Architecture Notes

This repository keeps the browser authentication flow intentionally simple:

- The login endpoint validates credentials and returns an access token.
- The server stores that token in an `HttpOnly` browser cookie named `access_token`.
- Protected backend endpoints read the token from the cookie and validate it with the existing JWT middleware.

### Why this approach

The goal is to keep the UI easy to use while keeping the implementation small and close to the current backend design.

- `HttpOnly` cookies reduce exposure of the token to JavaScript and make the browser handle session persistence automatically.
- The current codebase already issues JWT access tokens, so moving that token into a cookie is the smallest change.
- This keeps local development straightforward because the cookie is also allowed over plain `http`.

### Current tradeoffs

This is a pragmatic, early-stage choice rather than a final security architecture.

- CSRF protection is intentionally deferred for now.
- Cookie security flags are kept simple to support local development.
- If the app later needs stronger production-hardening, the auth flow can be extended with CSRF protection, HTTPS-only cookies, and refresh-token rotation or server-side sessions.

## Auth

Password based authentication - salt