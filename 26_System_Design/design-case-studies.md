# Design Case Studies

Four classic system design questions, each answered in the format real
interviews and architecture reviews use: requirements, a high-level
diagram, then a deep dive on the hardest 2-3 components — with every
piece mapped to the specific earlier module/project that already
implements it, so "design X" always ends with "and here's working code
for the hard part."

---

## Design YouTube

**Requirements**: users upload videos; videos are encoded into multiple
qualities; users watch videos with minimal buffering, worldwide;
view counts and recommendations update continuously.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Upload ──▶ Upload Service ──▶ Object Storage (raw video)          │
│                    │                                                   │
│                    ▼                                                     │
│              Message Queue (Module 20) ──▶ Transcoding Workers              │
│                                               (a WORKER POOL, Module 12,        │
│                                                encoding each upload into            │
│                                                multiple resolutions)                   │
│                    │                                                                      │
│                    ▼                                                                        │
│              Object Storage (encoded renditions) ──▶ CDN (this guide)                          │
│                                                              │                                     │
│   Viewer ──────────────────────────────────────────────────┘                                        │
│      (video bytes served from the NEAREST CDN edge, never the origin                                    │
│       for popular content — the origin is only hit on a CDN cache miss)                                    │
│                                                                                                              │
│   View counts: an EVENT (Module 22's Event Sourcing) published on every                                       │
│   view, aggregated asynchronously (CQRS's read-model projector) rather                                          │
│   than an UPDATE on every single view hitting the video's row directly                                            │
└──────────────────────────────────────────────────────────┘
```

**Deep dive — why transcoding is a worker pool, not inline in the upload
request**: encoding a video into several resolutions takes real time
(minutes, for a long video) — doing it synchronously inside the upload
HTTP request (Module 15) would mean the uploader's connection stays open
that whole time, which is both a bad user experience and fragile (any
network hiccup loses the whole upload). Publishing an "upload received"
event and processing it via Module 12's worker-pool pattern (or Module 20's
message-queue consumer pattern) decouples "accept the upload" from "do the
expensive work," exactly the same shape as Module 21's async payment
gateway.

**Deep dive — view count consistency**: real-time-exact view counts
aren't actually needed (nobody notices "1,000,412" vs. "1,000,415" being a
few seconds stale) — this is a textbook case for eventual consistency
(Module 22), aggregating view events into a count periodically rather than
incrementing a single row on every view, which would become a massive
write bottleneck (a hot row, contended by every concurrent viewer) on
popular videos.

---

## Design Uber

**Requirements**: riders request rides; nearby drivers are matched;
locations update in real time during a trip; a fare is calculated and
charged at the end.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Rider App ──▶ Matching Service ──▶ (queries nearby drivers via a      │
│                                          geospatial index — Module 25's       │
│                                          "hard part" callout for this           │
│                                          exact project)                             │
│                     │                                                                  │
│                     ▼                                                                    │
│               Trip Service ──▶ publishes "trip.started" (Module 20,                        │
│                                   choreographed, not orchestrated — Rider                       │
│                                   App, Driver App, and Billing Service ALL                         │
│                                   react to the same event independently)                              │
│                     │                                                                                    │
│                     ▼                                                                                       │
│   Driver's live location: WebSocket (Module 25) streaming position                                            │
│   updates to the Rider App in real time — NOT polling, since a rider                                             │
│   watching a driver's car move on a map needs continuous updates, not                                               │
│   request/response                                                                                                    │
│                     │                                                                                                    │
│                     ▼                                                                                                       │
│   Trip ends ──▶ Payment Service (Module 21's fintech-backend fits                                                              │
│                    directly: charge the rider's wallet/card, credit the                                                           │
│                    driver, via a SAGA (Module 22) if it spans separate                                                                │
│                    services — charge fails → the trip's fare dispute flow                                                                │
│                    is the compensation)                                                                                                     │
└──────────────────────────────────────────────────────────┘
```

**Deep dive — why matching is choreographed, not one big transaction**:
"match a rider to a driver" touches multiple concerns (driver
availability, rider preferences, pricing/surge) that don't need — and
shouldn't have — one service owning all of it. A Saga-style flow (Module
22) where each concern is its own step, with a compensation if a later
step fails (the matched driver declines — compensate by re-opening the
search) fits far better than one large atomic operation.

**Deep dive — real-time location updates at scale**: millions of drivers'
positions updating every few seconds is fundamentally a Backpressure
problem (this guide's own section) — location updates that arrive faster
than downstream systems can process them need to be sampled/throttled
rather than queued without bound, since a slightly-stale location is
harmless but an unbounded queue of them is a real outage risk.

---

## Design WhatsApp

**Requirements**: users send messages to each other (and groups) in real
time; messages are delivered even if a recipient is offline; delivery and
read receipts are shown.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Sender ──WebSocket──▶ Chat Server (Module 25's chat-server, at            │
│                            genuinely larger scale — sharded across many          │
│                            server instances, one Hub per shard, matched              │
│                            to Module 26's Load Balancing section's                       │
│                            consistent-hashing approach so the SAME user's                   │
│                            connection always lands on the SAME shard)                          │
│        │                                                                                          │
│        ▼                                                                                            │
│   Message Queue (Module 20) — durable, so a message survives even if the                              │
│   recipient's Chat Server instance restarts before delivering it                                         │
│        │                                                                                                    │
│        ▼                                                                                                       │
│   Recipient ONLINE?  → deliver via their own open WebSocket connection                                            │
│   Recipient OFFLINE?  → hold in queue/database, PUSH NOTIFICATION sent,                                              │
│                           delivered once they reconnect                                                                 │
└──────────────────────────────────────────────────────────┘
```

**Deep dive — delivery guarantees**: a chat message is the textbook
at-least-once delivery case (Module 20/21) — losing a message is
unacceptable, so the system must be willing to occasionally deliver one
twice rather than risk dropping it, which means the CLIENT (not just the
server) needs idempotent handling — a message ID the receiving app
recognizes and deduplicates on, exactly Module 21's webhook-handling
pattern, applied to a chat bubble instead of a payment.

**Deep dive — group messages**: fanning one message out to every group
member is the exact same problem Module 25's Chat Server solved with its
Hub — broadcasting to N connected clients — except at WhatsApp's scale, a
single in-process Hub goroutine isn't enough; the real solution shards
group membership across many Chat Server instances and uses a message
queue (not a single Go channel) as the fan-out mechanism between them.

---

## Design Stripe

**Requirements**: merchants accept payments from customers; funds settle
to merchant accounts; the system must be auditable, idempotent, and
resilient to the actual payment networks (card networks, banks) being
slow or unreliable.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Merchant's website ──▶ Stripe API (Module 15/16's REST patterns,        │
│                              Module 18's API-key authentication)               │
│                    │                                                              │
│                    ▼                                                                │
│              Payment Gateway (Module 21's ENTIRE fintech-backend:                       │
│                                  the ledger, the idempotency-keyed webhook                  │
│                                  handling, the Circuit Breaker (this guide's                    │
│                                  project) wrapping every call OUT to a real                        │
│                                  card network/bank, since those are exactly                           │
│                                  the "unreliable downstream dependency" this                             │
│                                  guide's Circuit Breaker section exists for)                                │
│                    │                                                                                            │
│                    ▼                                                                                              │
│              Ledger (Module 21) — every charge, refund, and fee is a                                                 │
│              double-entry transaction, permanently recorded, reconciled                                                 │
│              continuously (Module 21's Reconciliation) against the real                                                    │
│              card networks' settlement reports                                                                               │
│                    │                                                                                                            │
│                    ▼                                                                                                              │
│              Webhook delivered to the MERCHANT's own server                                                                          │
│              (Module 21's exact webhook pattern, one more hop out —                                                                       │
│              Stripe's webhooks to merchants work IDENTICALLY to the fake                                                                     │
│              bank's webhooks to Stripe itself in Module 21's demo)                                                                              │
└──────────────────────────────────────────────────────────┘
```

**Deep dive — why idempotency keys are a first-class API concept, not an
implementation detail**: Stripe's real API requires clients to pass their
own `Idempotency-Key` header on every mutating request — this is Module
21's webhook idempotency pattern, exposed as a *contract* the client
opts into deliberately, specifically because a merchant's own network call
to Stripe can fail ambiguously (did the charge go through before the
timeout, or not?) and retrying blindly would risk a double charge. This is
the single most consequential API design decision in this whole case
study, and it's already fully implemented in Module 21's `webhook` package
— the same mechanism, just offered outward as a public contract instead
of used only internally.

**Deep dive — resilience to the actual card networks**: this is precisely
what this module's Circuit Breaker project exists to protect against — a
card network having a bad few minutes shouldn't take down Stripe's entire
API for every merchant; failing fast (once the breaker opens) and queuing
affected charges for retry once the network recovers is far better than
every merchant's checkout page hanging for 30 seconds per request.
