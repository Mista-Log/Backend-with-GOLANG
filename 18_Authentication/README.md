# Go for Beginners — Module 18: Authentication

## Contents

1. **[18-authentication.md](./18-authentication.md)** — JWT structure and
   verification (with a diagram of why JWTs scale without a database
   lookup, and the trade-off that creates for revocation), OAuth 2.0's
   Authorization Code flow (a full step-by-step diagram explaining *why*
   the code-then-exchange indirection exists), Google Login (the same
   protocol, Google's endpoints), refresh tokens with rotation and reuse
   detection (a theft-scenario diagram), RBAC, fine-grained permissions
   (with a direct RBAC-vs-permissions comparison), and sessions vs. tokens.
   Diagrams throughout every section.

2. **[authentication-service/](./authentication-service)** — A complete
   service exercising every section together: bcrypt registration/login,
   short-lived JWT access tokens, rotating opaque refresh tokens with real
   reuse detection (an actual theft scenario you can trigger and watch get
   caught), both RBAC (`requireRole`) and permission-based
   (`requirePermission`) middleware protecting different routes so the
   distinction is concrete, and a full OAuth2 Authorization Code flow
   against a **locally simulated provider** — the entire project runs
   offline, with real, unmodified `golang.org/x/oauth2` client code that
   would work against actual Google endpoints with only a config change.

## Suggested Order

```
Authentication guide ──▶ Authentication Service
```

One comprehensive project, deliberately — authentication's pieces
(hashing, tokens, rotation, authorization, OAuth) only really make sense as
one coherent flow, which is exactly what the service's README walks
through end to end with a curl command (and matching diagram) for every
stage, including the two most important **security properties to actually
witness working**: refresh token reuse detection catching a simulated theft,
and the OAuth state parameter rejecting a mismatched callback.

## Setup

```bash
cd authentication-service
go mod tidy    # fetches golang-jwt, golang.org/x/crypto, golang.org/x/oauth2
go run .
```

*Note: this module builds on Modules 00–17 — start there first if you
haven't already, especially Module 06 (interfaces — embedding
`jwt.RegisteredClaims` to satisfy `jwt.Claims`), Module 13 (context —
claims are propagated exactly the way Module 13 taught), and Module 15
(HTTP — the middleware chaining pattern this project's `authMiddleware` /
`requireRole` / `requirePermission` stack builds on directly).*
