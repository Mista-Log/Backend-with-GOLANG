# Go for Beginners — Module 25: Large Projects

*Build these from scratch.*

## Contents

1. **[25-large-projects.md](./25-large-projects.md)** — A blueprint for
   all 18 named projects across Beginner, Intermediate, Advanced, and
   Expert tiers: scope, starting data model, which earlier module's
   patterns apply directly, and the one genuinely hard part to plan for
   before starting each one. Closes with a WebSockets primer, the one
   piece of transport this entire course hadn't covered yet.

2. **[cli-todo/](./cli-todo)** (Beginner) — A full, crash-safe CLI task
   manager using Module 10's temp-file-then-rename safe-write pattern.

3. **[chat-server/](./chat-server)** (Intermediate) — A full real-time
   WebSocket chat server, solving the concurrent-broadcast problem with a
   single Hub goroutine (Module 12's "share memory by communicating"
   proverb) instead of a Mutex-protected map.

## Why Only Two Are Built in Full

The Advanced tier's Banking System, Payment Gateway, and Wallet are
already built, in real depth, as Module 21's `fintech-backend` — rebuilding
them here would just be less-complete duplicates of work you already have.
The Expert tier's six "clone" projects are explicitly framed in the guide
as **combinations** of that same Module 21 foundation plus a public API
layer (Modules 15/16/20), real persistence (Module 17), and deployment
(Module 24) — there's no new pattern in any of them this course hasn't
already covered somewhere. Building CLI Todo and Chat Server instead means
every project in this module teaches something, rather than three of them
teaching the same lesson twice.

## Setup

```bash
cd cli-todo && go run . add "Buy milk" && go run . list

cd ../chat-server && go mod tidy && go run .
# open http://localhost:8080/?name=ada and ?name=bob in two tabs
```

---

This is the last module in the course. Every topic from `go version` in
Module 00 through this roadmap has been building toward exactly this: the
ability to look at an unfamiliar, real-world system and know which pieces
you already know how to build.
