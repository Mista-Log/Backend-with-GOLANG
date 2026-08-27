# 18. Authentication

Every project so far has been open to anyone who can reach it. This module
covers proving *who* is making a request (authentication) and deciding
*what they're allowed to do* (authorization) — the two concerns that sit in
front of almost every real API.

---

## JWT

A **JSON Web Token** is a compact, self-contained, cryptographically signed
token — the server can verify it wasn't tampered with **without any
database lookup**, because the signature itself proves authenticity.

```
┌──────────────────────────────────────────────────────────┐
│                    A JWT has THREE parts, dot-separated               │
│                                                                            │
│   eyJhbGciOiJIUzI1NiJ9 . eyJzdWIiOiI0MiIsInJvbGUiOiJhZG1pbiJ9 . SflKxwRJ  │
│   ─────────┬─────────    ─────────────┬─────────────────────   ────┬────  │
│           HEADER                    PAYLOAD                     SIGNATURE   │
│   {"alg":"HS256"}         {"sub":"42","role":"admin",          HMAC-SHA256  │
│                             "exp":1234567890}                   of header+   │
│                                                                    payload,      │
│   Both header and payload are just base64 — ANYONE can decode         signed        │
│   and READ them (never put secrets in a JWT payload!). Only the        with a         │
│   SIGNATURE proves the server actually issued it, unmodified.           SECRET KEY       │
└──────────────────────────────────────────────────────────┘
```

```go
import "github.com/golang-jwt/jwt/v5"

// Issuing a token:
claims := jwt.MapClaims{
	"sub":  "42",              // subject — conventionally the user ID
	"role": "admin",
	"exp":  time.Now().Add(15 * time.Minute).Unix(), // expiration — ALWAYS set one
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signed, err := token.SignedString([]byte("your-secret-key"))

// Verifying a token:
parsed, err := jwt.Parse(signed, func(t *jwt.Token) (any, error) {
	return []byte("your-secret-key"), nil
})
if err != nil || !parsed.Valid {
	// reject — invalid signature, expired, or malformed
}
claims := parsed.Claims.(jwt.MapClaims)
userID := claims["sub"]
```

```
┌──────────────────────────────────────────────────────────┐
│    Why JWTs scale well: NO database round-trip to verify one!         │
│                                                                            │
│    Session-based (Section "Sessions" below): every request needs           │
│    a database/cache lookup to check if the session is still valid            │
│                                                                                  │
│    JWT-based: the SIGNATURE itself is the proof — verify it                       │
│    mathematically, in-memory, no external lookup needed at all                        │
│                                                                                            │
│    The trade-off: a JWT can't be "revoked" before it expires without                        │
│    SOME extra mechanism (a blocklist, short expiry + refresh tokens —                          │
│    see below) — the token remains valid, cryptographically, until its                             │
│    stated expiry, no matter what happens server-side in the meantime.                                │
└──────────────────────────────────────────────────────────┘
```

**Always set a short expiration on access tokens** (minutes, not days) —
this bounds exactly how long a stolen token remains useful, which is the
whole reason Refresh Tokens (below) exist as a companion mechanism.

---

## OAuth

**OAuth 2.0** is a protocol for **delegated authorization** — letting a
user grant your application limited access to their account on *another*
service (Google, GitHub, ...) without ever handing your application their
password for that service. The most common flow is the **Authorization
Code flow**:

```
┌──────────────────────────────────────────────────────────────┐
│    YOUR APP              THE USER'S BROWSER            OAUTH PROVIDER      │
│                                                              (e.g. Google)     │
│      │                          │                                │              │
│      │  1. redirect to provider's                                │              │
│      │     /authorize?client_id=...&redirect_uri=...             │              │
│      │─────────────────────────▶│                                │              │
│      │                          │  2. user logs in & approves        │              │
│      │                          │─────────────────────────────────▶│              │
│      │                          │                                │              │
│      │                          │  3. redirect BACK to YOUR redirect_uri,          │
│      │                          │     with a one-time AUTHORIZATION CODE             │
│      │                          │◀─────────────────────────────────│              │
│      │◀─────────────────────────│                                │              │
│      │                          │                                │              │
│      │  4. YOUR SERVER exchanges the code for tokens                                │
│      │     (this happens SERVER-TO-SERVER, not through the browser)                    │
│      │────────────────────────────────────────────────────────▶│              │
│      │◀────────────────────────────────────────────────────────│              │
│      │       access_token, (id_token, refresh_token, ...)                              │
│      │                                                                                    │
│      │  5. YOUR SERVER calls the provider's userinfo endpoint                                │
│      │     with the access_token to learn who the user actually is                             │
│      │────────────────────────────────────────────────────────▶│              │
│      │◀────────────────────────────────────────────────────────│              │
│      │       {"email": "user@example.com", "name": "..."}                                  │
└──────────────────────────────────────────────────────────────┘
```

**Why the code-then-exchange indirection**, instead of just returning the
access token directly in step 3: the browser redirect in step 3 is visible
in browser history, server logs, and the Referer header — a genuine access
token there would be exposed to anyone with access to those. The
**authorization code** is short-lived and single-use, and the actual token
exchange (step 4) happens over a direct, authenticated server-to-server
connection using a **client secret** only your server knows — the access
token itself never appears anywhere the browser can see it.

```go
import "golang.org/x/oauth2"

conf := &oauth2.Config{
	ClientID:     "your-client-id",
	ClientSecret: "your-client-secret",
	RedirectURL:  "https://yourapp.com/oauth/callback",
	Scopes:       []string{"email", "profile"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://provider.example.com/authorize",
		TokenURL: "https://provider.example.com/token",
	},
}

// Step 1: redirect the user here
authURL := conf.AuthCodeURL("random-state-value") // "state" prevents CSRF — verify it matches on callback

// Step 4: exchange the code your callback handler received
token, err := conf.Exchange(ctx, code)
```

---

## Google Login

"Sign in with Google" is OAuth's Authorization Code flow, specifically
against Google's endpoints, via `golang.org/x/oauth2/google`:

```go
import "golang.org/x/oauth2/google"

conf := &oauth2.Config{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	RedirectURL:  "https://yourapp.com/oauth/google/callback",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint, // pre-filled with Google's real URLs
}
```

The mechanics are **identical** to generic OAuth above — `google.Endpoint`
just saves you from hand-typing Google's specific authorize/token URLs.
After exchanging the code, Google's `userinfo` endpoint (or decoding the
`id_token`, a JWT Google also returns) gives you the user's email and name,
which your application then maps to a **local user account** — creating
one on first login, or matching an existing one by email.

**This module's project simulates an OAuth provider locally** (via
`httptest`, the same technique used for external services throughout this
course) so the full flow — redirect, code exchange, userinfo fetch, local
account creation — runs completely offline, without real Google
credentials, while exercising the exact same protocol mechanics.

---

## Refresh Tokens

Since access tokens (JWTs) are deliberately short-lived, a **refresh
token** lets a client obtain a new access token without forcing the user to
log in again — typically a long-lived, opaque (non-JWT) random string,
stored server-side so it *can* be individually revoked (unlike a JWT).

```
┌──────────────────────────────────────────────────────────┐
│   Login  →  access token (15 min)  +  refresh token (30 days)         │
│                                                                            │
│   ... 16 minutes later, access token has expired ...                        │
│                                                                                  │
│   POST /refresh  {refreshToken: "..."}                                            │
│        │                                                                             │
│        ▼                                                                                │
│   server looks up the refresh token SERVER-SIDE (a real lookup,                            │
│   unlike verifying a JWT) — checks it exists, isn't expired, isn't                             │
│   already revoked/used                                                                            │
│        │                                                                                              │
│        ▼                                                                                                 │
│   issues a NEW access token AND a NEW refresh token, INVALIDATING                                            │
│   the old refresh token — this is called ROTATION                                                               │
└──────────────────────────────────────────────────────────┘
```

**Rotation and reuse detection:** each refresh token is valid for exactly
**one** use. If a refresh token is presented a *second* time, that's a
strong signal it was stolen and used by someone else already — a
well-built system treats this as a security event and revokes the **entire
token family** (every token descended from that original login), not just
the one reused token, forcing a fresh login.

```
┌──────────────────────────────────────────────────────────┐
│   Legitimate flow:   RT1 used → RT2 issued (RT1 now dead)                │
│                       RT2 used → RT3 issued (RT2 now dead)                   │
│                                                                                  │
│   Theft scenario:    RT1 stolen by an attacker                                     │
│                       Legitimate user uses RT1 → RT2 issued (RT1 dead)                 │
│                       Attacker LATER tries RT1 → REJECTED (already used!)                 │
│                       → this reuse attempt REVOKES RT2 as well — the                          │
│                         legitimate user is forced to log in again, but                           │
│                         the attacker's stolen token is now completely dead                          │
└──────────────────────────────────────────────────────────┘
```

---

## RBAC

**Role-Based Access Control** — users are assigned one or more **roles**
("admin," "editor," "viewer"), and access rules are defined in terms of
roles rather than individual users.

```go
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func requireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := claimsFromContext(r.Context()) // set by an earlier auth middleware
			if claims.Role != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

router.Handle("/admin/users", requireRole(RoleAdmin)(adminUsersHandler))
```

```
┌────────────────────────────────────────────────────┐
│   User "alice" ──▶ role: admin  ──▶ CAN access /admin/*    │
│   User "bob"    ──▶ role: user   ──▶ CANNOT access /admin/*  │
└────────────────────────────────────────────────────┘
```

---

## Permissions

RBAC alone becomes clunky once access rules get more granular than "admin
vs. not" — **permission-based** authorization checks specific capabilities
(`"products:write"`, `"orders:read"`) instead of a coarse role label, with
roles acting as **named bundles of permissions**:

```go
var rolePermissions = map[Role][]string{
	RoleUser:  {"orders:read"},
	RoleAdmin: {"orders:read", "orders:write", "products:read", "products:write"},
}

func requirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := claimsFromContext(r.Context())
			for _, p := range rolePermissions[claims.Role] {
				if p == permission {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
```

```
┌──────────────────────────────────────────────────────────┐
│                     RBAC              vs.        Permissions              │
│                                                                                │
│   "is this user an admin?"          "can this user WRITE PRODUCTS,             │
│                                        specifically?"                             │
│                                                                                        │
│   Coarse — adding a new capability   Finer-grained — a new "moderator"                  │
│   often means creating a whole new     role can be assembled from EXISTING                 │
│   role, or overloading an existing     permissions, no new special-case                       │
│   one with unrelated meaning             logic needed anywhere else                              │
└──────────────────────────────────────────────────────────┘
```

Most real systems use **both together**, exactly as sketched above: roles
for simplicity at the user-management level ("make Bob an editor"),
permissions underneath for the actual authorization checks in code.

---

## Sessions

The classic alternative to token-based auth: after login, the server
creates a **session** (server-side state — who's logged in, when it
expires) and gives the browser a **session ID** via a cookie (Module 15).
Every subsequent request, the server looks up that ID to find the session.

```
┌──────────────────────────────────────────────────────────┐
│              Session-based                vs.        Token-based (JWT)      │
│                                                                                  │
│   Server stores session state             Server stores NOTHING extra —           │
│   (in memory, Redis, a database)            the token itself is self-contained       │
│                                                                                          │
│   Every request needs a lookup            Every request verified via SIGNATURE         │
│   to check session validity                  alone — no lookup needed                     │
│                                                                                                │
│   Revoking access: DELETE the             Revoking access: needs a blocklist or             │
│   session server-side — instant             short expiry + refresh tokens                       │
│                                                                                                        │
│   Natural fit for traditional            Natural fit for APIs, mobile apps,                        │
│   server-rendered web apps                  and services calling OTHER services                       │
└──────────────────────────────────────────────────────────┘
```

**Neither is universally "better"** — sessions make revocation trivial but
need shared server-side storage (a real cost at scale, across multiple
server instances); JWTs scale beautifully (no shared storage needed to
verify one) but make revocation harder, which is exactly the gap refresh
tokens exist to close. This module's project uses JWTs for access + a
server-side-tracked refresh token specifically to get *both* benefits: fast,
stateless verification most of the time, with a real, revocable
server-side record for the one operation (refreshing) that needs it.

---

Onto the project — Authentication Service ties every section together:
registration with hashed passwords, JWT access tokens, rotating refresh
tokens with reuse detection, RBAC and permission-based middleware, and a
full OAuth login flow against a locally simulated provider.
