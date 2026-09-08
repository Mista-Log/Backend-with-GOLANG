# Go for Beginners — Module 21: FinTech Backend

*This is where you become industry-ready.*

## Contents

1. **[21-fintech-backend.md](./21-fintech-backend.md)** — Double-entry
   accounting (why debit/credit don't mean subtract/add, and which
   direction increases which account type), the ledger as the single,
   append-only source of truth, wallets, transactions, settlement,
   reconciliation, payment gateways, escrow, webhooks, idempotency, fraud
   detection, KYC, AML, PCI DSS (and why almost no backend engineer ever
   touches a raw card number), the two distinct kinds of audit trail a
   real system needs, multi-currency accounting, and exchange rates.
   Diagrams throughout every section.

2. **[fintech-backend/](./fintech-backend)** — One integrated system
   implementing all nine named projects as real modules on a shared
   double-entry ledger: **Wallet System, Transfer API, Virtual Accounts,
   Payment Gateway, Bank API, Ledger Engine, Escrow System, Savings App,**
   and **Loan Engine** — plus Fraud/KYC/AML checks, multi-currency FX,
   reconciliation, and audit logging. Its own README documents three real
   accounting bugs this project's own rigor caught during development
   (a wrong account type, a subtle "arithmetically balanced but
   economically backwards" bug the ledger's own balance check couldn't
   catch, and a missing balance check) — left in, with the fixes, because
   that honesty is more valuable than a pretend-perfect first draft.

## Why One Project, Not Nine

Every real fintech backend has this exact shape: one ledger at the center,
many products built on top of it. Treating Wallet System, Payment Gateway,
Transfer API, Bank API, Virtual Accounts, Ledger Engine, Savings App, Loan
Engine, and Escrow System as nine unrelated apps would have been a less
honest capstone than building the one integrated system a real fintech
engineering team actually ships.

## Setup

```bash
cd fintech-backend
go run ./cmd/demo
go test ./...
```

No external dependencies — pure standard library, so this runs immediately
with no setup step at all.

*Note: this module draws on essentially everything in this course —
Module 06 (interfaces, the `RateProvider` and repository-style ports),
Module 08 (errors — every check returns a real, wrapped error), Module 12
(concurrency — every service is `Mutex`-protected for concurrent access),
Module 13 (context, in the FX conversion path), Module 15/18/20 (the
webhook receiver's real HTTP server, and the fake bank built the same way
Module 18 built a fake OAuth provider), and Module 19 (the `internal/`
layout and ports-and-adapters structure this project's module boundaries
follow directly). If this is the last module you're working through, it's
meant to feel like everything else finally clicking into one place.*
