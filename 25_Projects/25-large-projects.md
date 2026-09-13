# 25. Large Projects

*Build these from scratch.*

This module is different from every one before it: not a topic to learn,
but a **roadmap** — 18 real projects across four tiers, each a genuine
synthesis of everything this course has covered. For every project below:
a scope, a starting data model, which earlier modules' patterns apply
directly, and the specific hard part worth planning for before you start.

Two projects are built out fully in this module (**CLI Todo** and **Chat
Server**) — chosen specifically because they add something genuinely new
(a real CLI persistence pattern, and WebSockets, which nothing earlier in
this course has covered). The rest point to exactly which earlier
module's project already demonstrates the hard part, since several of
these tiers substantially overlap with work you've already built.

---

## Beginner

### CLI Todo

A command-line task manager with persistent storage between runs — no
server, no database engine, just a program and a file.

```
┌────────────────────────────────────────────────────┐
│   Data model:  Task{ID, Title, Done, CreatedAt}                       │
│   Storage:      a JSON file on disk (Module 10/11)                       │
│   Interface:      subcommands via os.Args or the flag package                │
│                     (Module 00), or cobra (Module 19) for a nicer CLI            │
│                                                                                      │
│   Hard part: making writes CRASH-SAFE. If the program is killed mid-                  │
│   write, the todo file shouldn't end up corrupted/truncated — this is                    │
│   Module 10's safe-write (temp file + atomic rename) pattern, applied                       │
│   to a CLI tool instead of a server.                                                            │
└────────────────────────────────────────────────────┘
```

**Built in full below** — see `cli-todo/`.

### Notes App

A richer CLI Todo, essentially — free-text notes instead of structured
tasks, usually with tagging and full-text search.

```
┌────────────────────────────────────────────────────┐
│   Data model:  Note{ID, Title, Body, Tags []string, CreatedAt}          │
│   Storage:      JSON file (small scale) or SQLite (Module 17) once            │
│                   search/filtering needs real querying                            │
│                                                                                        │
│   Hard part: SEARCH. A linear substring scan (Module 16's Search                        │
│   section) is fine until the note count gets large — SQLite's FTS5                         │
│   extension (full-text search) is the natural next step, and a good                           │
│   reason to revisit Module 17's sqlx patterns.                                                    │
└────────────────────────────────────────────────────┘
```

### URL Shortener

Given a long URL, generate a short code; given the short code, redirect
back to the original.

```
┌────────────────────────────────────────────────────┐
│   Data model:  ShortURL{Code, OriginalURL, CreatedAt, ClickCount}       │
│   Storage:      a database (Module 17) — this needs real persistence,        │
│                   since a restart shouldn't lose every mapping                       │
│   Interface:      REST API (Module 15/16): POST /shorten, GET /{code}              │
│                     (a 302 redirect)                                                    │
│                                                                                              │
│   Hard part: CODE GENERATION under concurrency. Two requests generating                        │
│   a short code at the SAME instant must never collide — either use a                              │
│   database-generated sequence/UUID (simplest, safest) or a base62-                                    │
│   encoded auto-increment ID, and reason carefully about the race                                          │
│   (Module 12) if you roll your own random-code-plus-retry-on-collision                                        │
│   scheme instead.                                                                                                │
└────────────────────────────────────────────────────┘
```

---

## Intermediate

### Blog API

A CRUD API for posts, authors, and comments, usually the first project
where authentication and authorization actually matter end to end.

```
┌────────────────────────────────────────────────────┐
│   Data model:  Post{ID, AuthorID, Title, Body, PublishedAt},                  │
│                  Comment{ID, PostID, AuthorID, Body}                              │
│   Storage:      Module 17's sqlx + repository pattern fits perfectly                 │
│   Interface:      REST API (Module 15/16): CRUD + pagination + search                    │
│   Auth:           Module 18 — JWT for the API, RBAC for "only the                            │
│                     author or an admin can edit/delete a post"                                  │
│                                                                                                       │
│   Hard part: AUTHORIZATION at the RESOURCE level, not just the route                                    │
│   level. "Is this user logged in" (Module 18's authMiddleware) is                                          │
│   necessary but not sufficient — "is this user allowed to edit THIS                                            │
│   SPECIFIC post" needs a check inside the handler itself, comparing the                                            │
│   authenticated user's ID against the post's AuthorID.                                                                │
└────────────────────────────────────────────────────┘
```

### Chat Server

Real-time, bidirectional messaging between many connected clients.

```
┌────────────────────────────────────────────────────┐
│   Transport: WebSockets — genuinely new to this course; see the             │
│                guide section below and the full build in chat-server/            │
│   Hard part: BROADCAST FAN-OUT safely. Every connected client is its              │
│                own goroutine (Module 12); broadcasting a message to                    │
│                ALL of them concurrently, while clients connect and                        │
│                disconnect at arbitrary moments, is a genuine concurrency                     │
│                design problem — solved below with a central Hub                                 │
│                goroutine that owns the client registry exclusively.                                │
└────────────────────────────────────────────────────┘
```

**Built in full below** — see `chat-server/`.

### Inventory System

Already built, in depth, as Module 04's project — stock levels, embedding
for perishable vs. regular products, low-stock reporting. A larger version
adds: Module 17's real database persistence (instead of in-memory), Module
16's filtering/sorting/search over the catalog, and Module 20's
choreography if inventory changes need to notify other services (an order
system reserving stock, for instance).

### File Storage

An API for uploading, storing, and retrieving files — think a minimal S3.

```
┌────────────────────────────────────────────────────┐
│   Data model:  FileMetadata{ID, Filename, Size, ContentType,           │
│                  StoragePath, UploadedAt}                                  │
│   Storage:      Module 10's file handling for the actual bytes                │
│                   (streamed, never loaded fully into memory — Module               │
│                   10's io.Copy pattern matters a LOT here for large files),           │
│                   Module 17's database for metadata                                     │
│   Interface:      REST API: POST /files (multipart upload),                                │
│                     GET /files/{id} (streamed download)                                       │
│                                                                                                    │
│   Hard part: STREAMING large uploads/downloads without loading the                                   │
│   whole file into memory. `r.Body` is already a stream (Module 15);                                     │
│   the mistake to avoid is calling io.ReadAll on it for a large upload —                                    │
│   stream directly to disk with io.Copy instead, exactly Module 10's habit.                                    │
└────────────────────────────────────────────────────┘
```

---
## Advanced

Four of these six projects are direct extensions of work you've already
built in full — the blueprint for each names exactly which existing
project to start from.

### Banking System

**Already built in full** — Module 21's `fintech-backend` IS a banking
system: double-entry ledger, accounts, transfers, reconciliation. Extend
it with real persistence (Module 17's SQLite/sqlx, replacing the in-memory
maps) and a real REST/gRPC API surface (Modules 15/16/20) in front of the
existing service layer — the business logic itself needs no changes,
exactly per Module 19's ports-and-adapters promise.

### Payment Gateway

**Already built in full** — Module 21's `paymentgateway` package: async
bank settlement, real webhooks, idempotent redelivery handling. Extend it
with real card tokenization concepts (Module 21's PCI DSS section — never
store raw card data, integrate a real provider's tokenization API
conceptually) and multiple payment methods routed through the same
interface (Module 06's polymorphism, exactly like Module 06's own Payment
Gateway project).

### Wallet

**Already built in full** — Module 21's `wallet` package, plus
`virtualaccounts` for external funding. Extend it with multi-wallet
support per user (Module 21's multi-currency section) and spending limits/
controls (a compliance check, Module 21's `fraud`/`kyc` pattern, applied
to "does this withdrawal exceed the user's configured daily limit").

### POS Backend

A point-of-sale system: register sales, apply tax/discounts, manage
shifts and till reconciliation.

```
┌────────────────────────────────────────────────────┐
│   Data model:  Sale{ID, Items []LineItem, Tax, Discount, Total,           │
│                  PaymentMethod, CashierID, ShiftID}                          │
│   Core logic:   this is a LEDGER problem — Module 21's double-entry               │
│                   principle applies directly: a sale is a transaction                │
│                   moving inventory OUT and cash/receivable IN, and a                    │
│                   till reconciliation at shift's end is EXACTLY Module 21's                │
│                   Reconciliation pattern (expected cash vs. actual cash drawer count)         │
│                                                                                                    │
│   Hard part: OFFLINE-FIRST operation. Real POS systems must keep taking                             │
│   sales even if the network drops — this needs local persistence with                                  │
│   later sync, a genuinely hard distributed-systems problem (Module 22's                                    │
│   consistency/replication sections apply directly to "what happens when                                       │
│   two offline registers both sold the last unit of the same item").                                              │
└────────────────────────────────────────────────────┘
```

### Ride Sharing Backend

Matching riders to drivers, tracking trips, calculating fares.

```
┌────────────────────────────────────────────────────┐
│   Services: Rider Service, Driver Service, Matching Service, Trip             │
│               Service, Payment Service — a genuine microservices system              │
│               (Module 20's exact shape: independent services, an event-                 │
│               driven flow — "ride requested" → matched → "trip started" →                  │
│               "trip completed" → payment charged)                                             │
│                                                                                                    │
│   Hard part: MATCHING under real-time constraints — finding the nearest                             │
│   available driver is a geospatial query most relational databases                                     │
│   handle poorly at scale; real systems use geospatial indexes (PostGIS,                                    │
│   or a geohashing scheme) — a good excuse to go deeper into Module 17's                                       │
│   database section for a feature not otherwise covered in this course.                                          │
└────────────────────────────────────────────────────┘
```

### Logistics API

Package tracking and delivery routing — conceptually a state machine
(`created → picked_up → in_transit → out_for_delivery → delivered`) with
an event trail.

```
┌────────────────────────────────────────────────────┐
│   Core logic: this is EVENT SOURCING (Module 22) in its purest form —          │
│                 a package's current status is DERIVED by replaying every           │
│                 tracking event ever recorded for it, and the full history               │
│                 (Module 22's Event Sourcing section) is exactly what a                     │
│                 "track my package" UI needs to show                                           │
│                                                                                                    │
│   Hard part: hooking up REAL webhook-driven updates from carriers                                    │
│   (Module 21's webhook + idempotency patterns, applied to "the carrier's                                │
│   system tells you a package moved," not a payment settling)                                               │
└────────────────────────────────────────────────────┘
```

---

## Expert — "Clone" Projects

Cloning Stripe, PayPal, Flutterwave, Moniepoint, Wise, or Revolut isn't
about matching their scale — it's about combining **every** Advanced-tier
concept into one coherent platform, the way each of those companies
actually does.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Stripe / PayPal / Flutterwave  ≈  Payment Gateway + Wallet +      │
│                                        Ledger Engine + Webhooks +       │
│                                        a merchant-facing REST API           │
│                                        (Module 21's WHOLE fintech-backend,      │
│                                        with a public API surface added)             │
│                                                                                          │
│   Moniepoint                      ≈  the above + POS Backend + Virtual                     │
│                                        Accounts (Module 21 already has                        │
│                                        this exact module) + agent banking                        │
│                                        (multi-tenant accounts, one operator                         │
│                                        managing many customer wallets)                                 │
│                                                                                                             │
│   Wise / Revolut                    ≈  the above + heavy Multi-Currency/                                     │
│                                          Exchange Rate emphasis (Module 21's                                    │
│                                          fx package) + real-time rate feeds                                        │
│                                          (would need a real market-data API,                                          │
│                                          Module 11's API-client patterns) +                                             │
│                                          card issuing (PCI DSS boundary,                                                   │
│                                          Module 21's guide section — this is                                                 │
│                                          the ONE piece you'd genuinely                                                          │
│                                          integrate a licensed provider for,                                                       │
│                                          never build yourself)                                                                       │
└──────────────────────────────────────────────────────────┘
```

**The realistic starting point for any of these six**: fork Module 21's
`fintech-backend`, add a real database (Module 17) in place of the
in-memory ledger, add a public REST/gRPC API (Modules 15/16/20) in front
of the existing service layer, add real authentication for merchants/API
keys (Module 18), containerize and deploy it (Module 24), and add the
product-specific feature each company is actually known for (Moniepoint's
agent banking, Wise's multi-currency accounts, Stripe's developer-first
API design). The ledger core underneath barely needs to change at all —
which is, again, exactly the point of building it as one shared foundation
in the first place.

---

## WebSockets — the One Genuinely New Piece

Everything above leans on patterns this course already covered, except
**Chat Server**'s transport. A WebSocket is a **persistent, full-duplex**
connection — unlike a normal HTTP request/response, either side can send a
message to the other **at any time**, without polling.

```
┌──────────────────────────────────────────────────────────┐
│   Normal HTTP:        client asks → server answers → connection CLOSES     │
│                          (need a NEW request for every new piece of info)       │
│                                                                                     │
│   WebSocket:            client and server UPGRADE one HTTP connection into           │
│                            a persistent, bidirectional pipe — EITHER side can           │
│                            send a message at ANY time, no new request needed               │
└──────────────────────────────────────────────────────────┘
```

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil) // the ONE HTTP request that starts the connection
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage() // blocks until the CLIENT sends something
		if err != nil {
			break // connection closed
		}
		conn.WriteMessage(websocket.TextMessage, message) // send something back, any time
	}
}
```

Every connected client is naturally its own goroutine (Module 12) reading
its own connection in a loop — the hard part, solved in `chat-server/`
below, is safely **broadcasting** one message out to every OTHER
connected client's goroutine at once.
