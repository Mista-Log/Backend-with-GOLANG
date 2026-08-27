# Project — Authentication Service

A complete authentication and authorization service covering every section
of Module 18: bcrypt password hashing, JWT access tokens, rotating refresh
tokens with theft detection, RBAC, fine-grained permissions, and a full
OAuth2 login flow — against a **locally simulated provider**, so the entire
project runs offline with zero real Google credentials needed.

## Setup

```bash
cd authentication-service
go mod tidy    # fetches golang-jwt, golang.org/x/crypto, golang.org/x/oauth2
go run .
```

---

## 1. Registration and Login

```bash
curl -X POST http://localhost:8080/register \
  -d '{"username": "kemi", "password": "hunter2000", "email": "kemi@example.com"}'

curl -X POST http://localhost:8080/login \
  -d '{"username": "kemi", "password": "hunter2000"}'
```

```
┌──────────────────────────────────────────────────────────┐
│   POST /register  {username, password, email}                       │
│        │                                                              │
│        ▼                                                                │
│   bcrypt.GenerateFromPassword(password)  ◀── the PLAINTEXT password       │
│        │                                       is hashed HERE and              │
│        ▼                                       NEVER stored anywhere              │
│   stored: PasswordHash = "$2a$10$N9qo8uLOickgx2Z..."                                  │
│                                                                                           │
│   POST /login  {username, password}                                                        │
│        │                                                                                       │
│        ▼                                                                                          │
│   bcrypt.CompareHashAndPassword(hash, password)  ◀── re-hashes the CANDIDATE,                        │
│        │                                              compares safely — the                            │
│        ▼                                              hash is NEVER reversed                              │
│   MATCH → issueTokenPair(user)                                                                                │
│        │                                                                                                         │
│        ▼                                                                                                            │
│   {"accessToken": "eyJhbGci...", "refreshToken": "a3f9c8e1..."}                                                        │
│         ▲                              ▲                                                                                │
│         │                              └── opaque random string, tracked SERVER-SIDE                                      │
│         └── a real JWT — decode it at jwt.io to see the header/payload/signature              │
│              structure from the guide, using this project's actual output                       │
└──────────────────────────────────────────────────────────┘
```

## 2. Using the Access Token

```bash
TOKEN="<paste the accessToken from login>"
curl http://localhost:8080/profile -H "Authorization: Bearer $TOKEN"
```
```
┌──────────────────────────────────────────────────────────┐
│   GET /profile                                                     │
│   Authorization: Bearer eyJhbGci...                                    │
│        │                                                                  │
│        ▼                                                                    │
│   authMiddleware:                                                              │
│     1. extract the token after "Bearer "                                          │
│     2. ParseAccessToken → verify SIGNATURE (no DB lookup at all!)                     │
│     3. check expiry (rejects if the 15-minute TTL has passed)                            │
│     4. context.WithValue(ctx, claimsContextKey, claims)  ◀── Module 13's                    │
│                                                                propagation habit               │
│        ▼                                                                                          │
│   profileHandler reads claims FROM THE CONTEXT, looks up the full user record                        │
│        │                                                                                                 │
│        ▼                                                                                                    │
│   200 OK  {"id":2,"username":"kemi","role":"user"}                                                             │
└──────────────────────────────────────────────────────────┘
```

## 3. RBAC vs. Permissions — Side by Side

```bash
# As "kemi" (role: user) — REJECTED by RBAC:
curl -i http://localhost:8080/admin/users -H "Authorization: Bearer $TOKEN"
# 403 Forbidden {"error":"requires role: admin"}

# Log in as the seeded admin instead:
curl -X POST http://localhost:8080/login -d '{"username":"admin","password":"adminpass123"}'
ADMIN_TOKEN="<paste accessToken>"

curl http://localhost:8080/admin/users -H "Authorization: Bearer $ADMIN_TOKEN"
# 200 OK — role check passed

curl -X POST http://localhost:8080/admin/products -H "Authorization: Bearer $ADMIN_TOKEN"
# 201 Created — PERMISSION check passed (admin's role grants "products:write")
```

```
┌──────────────────────────────────────────────────────────┐
│   GET /admin/users     ──▶ authMiddleware ──▶ requireRole(RoleAdmin)          │
│                                                     │                             │
│                                            claims.Role == "admin"?                    │
│                                              NO  → 403                                  │
│                                              YES → proceed                                 │
│                                                                                                │
│   POST /admin/products ──▶ authMiddleware ──▶ requirePermission("products:write")│
│                                                     │                                              │
│                                            rolePermissions["admin"] contains                          │
│                                            "products:write"?                                             │
│                                              NO  → 403                                                      │
│                                              YES → proceed                                                     │
│                                                                                                                    │
│   Both routes are protected, but by DIFFERENT checks — try adding a new                                              │
│   "editor" role in rbac.go that gets "products:write" but NOT the full                                                  │
│   admin role, and confirm which of these two routes it can reach.                                                          │
└──────────────────────────────────────────────────────────┘
```

## 4. Refresh Token Rotation — the Normal Path

```bash
RT="<paste the refreshToken from login>"
curl -X POST http://localhost:8080/refresh -d "{\"refreshToken\": \"$RT\"}"
```
```
┌──────────────────────────────────────────────────────────┐
│   RefreshTokenStore.Rotate(RT1)                                       │
│        │                                                                 │
│        ▼                                                                   │
│   record found, unused, unexpired  ──▶  mark RT1 USED                        │
│        │                                                                        │
│        ▼                                                                           │
│   issue RT2, SAME family as RT1                                                       │
│        │                                                                                 │
│        ▼                                                                                    │
│   {"accessToken": "<new JWT>", "refreshToken": "<RT2>"}                                          │
│                                                                                                       │
│   RT1 is now DEAD — using it again will trigger reuse detection (next section).                        │
└──────────────────────────────────────────────────────────┘
```

## 5. Refresh Token Reuse Detection — the Theft Scenario, Demonstrated

```bash
# RT1 was already used above to get RT2. Try using RT1 AGAIN:
curl -i -X POST http://localhost:8080/refresh -d "{\"refreshToken\": \"$RT\"}"
# 401 Unauthorized {"error":"refresh token reuse detected — entire session family revoked"}

# Now try RT2 (the token issued FROM that legitimate rotation) — also dead:
curl -i -X POST http://localhost:8080/refresh -d "{\"refreshToken\": \"<RT2>\"}"
# 401 Unauthorized {"error":"unknown refresh token"}
```
```
┌──────────────────────────────────────────────────────────┐
│   Timeline:                                                             │
│                                                                              │
│   t=0   Login           → RT1 issued (family F)                               │
│   t=1   Refresh(RT1)     → RT1 marked used, RT2 issued (family F)                 │
│   t=2   Refresh(RT1) AGAIN → RT1.used == true → REUSE DETECTED                       │
│                              → revokeFamilyLocked(F) → RT2 ALSO deleted                 │
│   t=3   Refresh(RT2)       → "unknown refresh token" (it no longer exists at ALL)         │
│                                                                                                │
│   This is exactly the guide's theft diagram: whoever holds RT1 at t=2 is                        │
│   an ATTACKER (the legitimate holder already moved on to RT2 at t=1) —                             │
│   and the response to that attempted reuse is to kill the WHOLE family,                               │
│   forcing anyone still using it (attacker or legitimate user alike) to                                   │
│   log in again from scratch.                                                                                │
└──────────────────────────────────────────────────────────┘
```

## 6. The Full OAuth Flow — One Command

```bash
curl -L http://localhost:8080/oauth/login
```

`-L` tells curl to **follow redirects** — which is worth doing once by hand
without `-L` first, to see each hop:

```bash
curl -i http://localhost:8080/oauth/login
# HTTP/1.1 302 Found
# Location: http://127.0.0.1:PORT/authorize?client_id=fake-client-id&...&state=...
```

```
┌──────────────────────────────────────────────────────────────┐
│   curl  ──▶  GET /oauth/login                                          │
│                    │                                                       │
│                    ▼  (this server)                                          │
│              generates `state`, redirects to the FAKE PROVIDER                  │
│                    │                                                               │
│                    ▼                                                                  │
│   curl  ──▶  GET <fake provider>/authorize?...&state=...                                 │
│                    │                                                                          │
│                    ▼  (fakeprovider — stands in for Google's login screen)                       │
│              "approves" instantly (see provider.go's comment on why),                               │
│              issues a one-time CODE, redirects back to OUR /oauth/callback                             │
│                    │                                                                                        │
│                    ▼                                                                                           │
│   curl  ──▶  GET /oauth/callback?code=...&state=...                                                              │
│                    │                                                                                                 │
│                    ▼  (this server, steps 4-5 of the guide's diagram)                                                   │
│              verify state matches ──▶ EXCHANGE code for an access token                                                    │
│              (server-to-server, POST to the fake provider's /token) ──▶                                                       │
│              fetch /userinfo with that access token ──▶ find-or-create a                                                         │
│              LOCAL user by email ──▶ issue OUR OWN access+refresh tokens                                                             │
│                    │                                                                                                                     │
│                    ▼                                                                                                                        │
│   curl sees the FINAL JSON response:                                                                                                           │
│   {"message": "logged in via OAuth as Ada Lovelace (ada@example.com)",                                                                            │
│    "tokens": {"accessToken": "...", "refreshToken": "..."}}                                                                                          │
└──────────────────────────────────────────────────────────────┘
```

Run it a second time — same email, so `FindOrCreateByEmail` matches the
**existing** local account instead of creating a duplicate:
```bash
curl -s -L http://localhost:8080/oauth/login | grep -o '"id":[0-9]*'
curl -s -L http://localhost:8080/oauth/login | grep -o '"id":[0-9]*'
# same user ID both times
```

## Case Study: What's Actually Fake Here, and What Isn't

```
┌──────────────────────────────────────────────────────────┐
│   FAKE (built for this demo, in fakeprovider/):                        │
│     - the /authorize endpoint auto-approving instead of showing            │
│        a real login form                                                     │
│     - the specific user "logged into" the provider (always the                 │
│        same one, KnownUser)                                                       │
│                                                                                        │
│   REAL (unmodified, would behave IDENTICALLY against actual Google):                    │
│     - golang.org/x/oauth2's Config, AuthCodeURL, and Exchange methods                       │
│     - the authorization code's single-use enforcement                                          │
│     - the state parameter CSRF check                                                              │
│     - the server-to-server code-for-token exchange                                                    │
│     - fetching user info with the resulting access token                                                  │
│     - the find-or-create-by-email account linking logic                                                       │
└──────────────────────────────────────────────────────────┘
```

Swapping this project to real Google login means changing exactly one
thing: `newOAuthConfig`'s `Endpoint` field, from the fake provider's URLs to
`google.Endpoint` (from `golang.org/x/oauth2/google`), plus using a real
Google Cloud Console client ID/secret. Every other line of `oauth.go` —
the state check, the exchange, the userinfo fetch, the account linking —
is already production-shaped.

## Try It Yourself
- Try refreshing with a refresh token that's simply made up (never issued
  at all) — confirm you get `"unknown refresh token"`, distinct from the
  reuse-detection message, and think through why that distinction might (or
  might not) matter for what you'd log or alert on in a real system
- Add a third role (`"editor"`) with a permission set that overlaps
  partially with admin's, and add a new route protected by a permission
  only editors and admins share — a good exercise in the guide's "roles as
  bundles of permissions" idea
- Add an `expires_in`-aware client-side check: modify the demo so a second
  `/profile` call, made after sleeping past the 15-minute access token TTL,
  correctly gets a 401 — then chain a `/refresh` call to recover, matching
  how a real client (a mobile app, a frontend) would handle an expired
  access token transparently
