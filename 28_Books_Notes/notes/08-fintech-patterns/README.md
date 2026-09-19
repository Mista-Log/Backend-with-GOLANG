# FinTech Architecture Patterns

Domain-specific architecture. The patterns here are less about Go and
more about correctness constraints money imposes — but Go's explicitness
suits them unusually well.

## Core patterns worth a note each

- Double-entry ledger as the single source of truth (Module 21)
- Idempotency keys as a public API contract, not an internal detail
- Money as integer minor units, never floats
- Escrow as a real account, not a status flag
- Reconciliation as a continuous process, not a monthly job
- The saga pattern for cross-service financial flows (Module 22)
- Outbox pattern for "update DB + publish event" atomicity (Module 22)

## Sources worth reading

- Stripe's engineering blog (especially on idempotency and API design)
- "Designing Data-Intensive Applications" ch. 7 (transactions)
- Martin Fowler's accounting patterns writing
- Public docs from Wise/Revolut/Monzo engineering blogs — Monzo's in
  particular are unusually detailed about ledger design

## Notes in this folder

- `money-as-integers.md`

## Related modules

Module 21 (the full fintech backend), Module 22 (saga, outbox), Module 17 (transactions)
