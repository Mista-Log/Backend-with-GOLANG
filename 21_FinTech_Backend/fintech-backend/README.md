# FinTech Backend — Capstone Project

One integrated system implementing all nine named projects as real modules
built on a shared double-entry ledger: **Wallet System, Transfer API,
Virtual Accounts, Payment Gateway, Bank API, Ledger Engine, Escrow System,
Savings App, and Loan Engine** — plus Fraud/KYC/AML checks, multi-currency
FX, reconciliation, and audit logging tying them all together.

This is deliberately **one system, not nine separate apps** — because
that's genuinely how real fintech backends are built. Stripe, Mono,
Brankas, and every serious payments company has exactly this shape: one
ledger at the center, many products sitting on top of it.

## Setup

```bash
cd fintech-backend
go run ./cmd/demo
go test ./...        # the ledger's own test suite — run this too
```

No external dependencies — everything is standard library, matching the
sandboxed, self-contained spirit of this project's earlier modules (this
one deliberately avoids Modules 16-20's `go mod tidy` step).

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                        │
│   Fraud/KYC/AML ──┐                                                     │
│                   ▼                                                       │
│              Transfer API ──┐                                                │
│                              │                                                  │
│   Wallet System ◀────────────┼──────────────┐                                    │
│         ▲                    │                │                                    │
│         │                    ▼                ▼                                     │
│         │              Escrow System    Virtual Accounts ──┐                          │
│         │                                                    │                          │
│         │              Savings App                            │                          │
│         │                    │                                  │                          │
│         │              Loan Engine                               │                          │
│         │                                                          │                          │
│         │              Payment Gateway ◀───────── Bank API ◀───────┘                          │
│         │                    │                                                                   │
│         └────────────────────┴──────────────────────┐                                             │
│                                                        ▼                                             │
│                                              ┌───────────────────┐                                     │
│                                              │   LEDGER ENGINE      │  ◀── everything, ultimately,          │
│                                              │  (double-entry,        │      goes through ONE Post()            │
│                                              │   append-only,           │      call on this ONE type               │
│                                              │   balance-checked)         │                                          │
│                                              └───────────────────┘                                     │
│                                                        │                                                 │
│                                                        ▼                                                   │
│                                                  Reconciliation                                              │
│                                                  Audit Log                                                     │
└──────────────────────────────────────────────────────────────────┘
```

Every arrow into the Ledger Engine is the SAME function call:
`ledger.Post(actor, idempotencyKey, description, entries)`. No module ever
mutates a balance directly — this is the whole point of double-entry
accounting, enforced structurally rather than by convention.

---

## Running the Demo — Section by Section

```bash
go run ./cmd/demo
```

### 1-3. Onboarding, Wallets, Deposits & Idempotency

```
┌──────────────────────────────────────────────────────────┐
│   kyc.Onboard("eve", "Jane Sanctioned")                              │
│        │                                                                │
│        ▼                                                                  │
│   sanctions watchlist match  →  BLOCKED before an account ever exists       │
│                                                                                 │
│   wallets.Deposit("teller", "dep-key-ada-1", "ada", 100000)                       │
│   wallets.Deposit("teller", "dep-key-ada-1", "ada", 100000)  ◀── SAME key, again    │
│        │                                                                              │
│        ▼                                                                                │
│   ledger.Post checks idempotency FIRST → returns the ORIGINAL transaction,               │
│   unchanged → ada's balance reflects ONE deposit, not two                                    │
└──────────────────────────────────────────────────────────┘
```

### 4-6. Transfer API, KYC Limits, Fraud Velocity

```
┌──────────────────────────────────────────────────────────┐
│   transfer.Send checks, IN ORDER:                                    │
│                                                                          │
│   1. kyc.CheckLimit    ◀── cheap, definitive, regulatory — checked FIRST    │
│   2. fraud.Check        ◀── heuristic, based on recent history                │
│   3. ledger.Balance      ◀── ground truth — checked LAST, right before Post      │
│   4. ledger.Post          ◀── the only step that can actually FAIL structurally    │
│                                                                                        │
│   bob (Basic tier, $1,000 limit) attempting $5,000  →  rejected at step 1,               │
│   never reaches fraud checking or the ledger at all                                         │
│                                                                                                  │
│   ada attempting 12 rapid transfers  →  passes KYC and the first ~9 fraud                          │
│   checks, then the 10-transaction velocity window trips  →  rejected at step 2                        │
└──────────────────────────────────────────────────────────┘
```

### 7. Multi-Currency / Exchange Rates

```
┌──────────────────────────────────────────────────────────┐
│   Convert($100 USD -> NGN)                                          │
│        │                                                               │
│        ▼                                                                 │
│   Leg 1: Credit wallet-ada (USD)     $100    [asset decreases]              │
│           Debit fx-clearing-USD       $100    [asset increases]               │
│        │                                                                         │
│        ▼  (rate applied: 1550.00)                                                  │
│   Leg 2: Credit fx-clearing-NGN      ₦155,000  [asset decreases]                       │
│           Debit wallet-ada-ngn        ₦155,000  [asset increases]                        │
│                                                                                               │
│   TWO separate, EACH individually balanced transactions — never one                            │
│   transaction mixing USD and NGN amounts directly (the guide's Multi                              │
│   Currency section's core rule).                                                                    │
└──────────────────────────────────────────────────────────┘
```

### 8-9. Virtual Accounts and Payment Gateway (real async webhooks)

```
┌──────────────────────────────────────────────────────────┐
│   gateway.InitiateDeposit("ada", $250)                               │
│        │  POSTs to the fake bank's real HTTP server                      │
│        ▼                                                                    │
│   bank responds 202 Accepted IMMEDIATELY — no ledger entry yet                 │
│        │                                                                          │
│        │  (150ms later, in the bank's own goroutine)                                │
│        ▼                                                                              │
│   bank POSTs a webhook to OUR real HTTP server: "deposit.settled"                        │
│        │                                                                                    │
│        ▼                                                                                      │
│   httpapi's handler → gateway.HandleWebhook → idempotency check → ledger.Post                    │
│        │                                                                                            │
│        ▼                                                                                              │
│   ada's balance updates — ONLY NOW, after real confirmation                                              │
│                                                                                                              │
│   Then: the EXACT SAME webhook payload is redelivered (simulating a real                                       │
│   gateway's at-least-once guarantee) → idempotency key already seen →                                            │
│   HandleWebhook returns immediately, doing NOTHING → balance UNCHANGED                                              │
└──────────────────────────────────────────────────────────┘
```

### 10-12. Escrow, Savings, Loans

Each of these is a different shape of the same core idea — moving money
between two Asset-typed ledger accounts the platform controls:

```
┌──────────────────────────────────────────────────────────┐
│   Escrow:   payer wallet  ⇄  escrow-holding  ⇄  recipient wallet         │
│   Savings:  spendable wallet  ⇄  locked savings account                     │
│   Loans:    loan-pool  ⇄  borrower wallet  (disbursement)                       │
│              borrower wallet  ⇄  loan-pool          (principal repayment)          │
│              borrower wallet  ⇄  interest-income      (interest repayment)            │
└──────────────────────────────────────────────────────────┘
```

### 13-14. Reconciliation and Audit Log

The demo runs THREE checks: an internal system-wide integrity check (must
always pass — if it doesn't, something is structurally broken), an
external check against a *matching* bank statement (passes), and one
against a **deliberately wrong** statement — proving the discrepancy
actually gets caught and reported, not just theoretically detectable.

---

## Case Studies: Three Real Bugs This Project's Own Rigor Caught

Building this project surfaced three genuine double-entry accounting bugs
— leaving them documented here, with the fixes, is more useful than
pretending the code was correct on the first attempt.

### Bug 1 — `savings.go`: wrong account type for interest

**The mistake:** modeling the interest counter-account as `Expense`.
Expense accounts increase on *Debit* — the exact same side as the Asset
savings account being credited with interest. Two accounts wanting to
increase on the same side can never balance in one transaction (`sum(debits)
== sum(credits)` becomes structurally impossible in the intended direction).

**The fix:** the interest counter-account needed to be `Liability`
("Interest Payable" — the platform now owes this to the customer), which
increases on *Credit* — correctly balancing against the savings account's
Debit-side increase.

### Bug 2 — `fx/rates.go`: correct arithmetic, wrong economics

**The mistake:** this one is subtler and more dangerous — the original
code's debit/credit directions were backwards (crediting the clearing
account and debiting the wallet, the reverse of what should happen), but
because *both* legs used matching amounts, `ledger.Post`'s balance check
**still passed**. The transaction was arithmetically balanced while moving
every wallet balance in the economically wrong direction.

```
┌──────────────────────────────────────────────────────────┐
│   This is the single most important lesson in this whole project:      │
│                                                                              │
│   The ledger's balance check guarantees the BOOKS balance.                    │
│   It does NOT guarantee a developer chose the correct SIDE for                   │
│   each account.                                                                      │
│                                                                                            │
│   Only correct accounting reasoning — or a reconciliation check                              │
│   noticing a balance moving the wrong way — catches THIS class of bug.                         │
└──────────────────────────────────────────────────────────┘
```

**The fix:** changed the clearing accounts to `Asset` type and swapped
each leg's debit/credit direction to match — see `fx/rates.go`'s comments
for the full reasoning.

### Bug 3 — `escrow.go`: a missing balance check

**The mistake:** `Create` posted the escrow-hold transaction without first
checking the payer actually had enough balance. Since `ledger.Post` only
enforces *debits == credits*, not *sufficient funds*, this would have let
a customer's wallet go negative.

**The fix:** added the same balance check `wallet.Withdraw` and
`transfer.Send` already had — a good reminder that this specific business
rule (no negative balances) has to be enforced consistently, by hand, at
every place money leaves an account; the ledger itself is deliberately
agnostic about it.

---

## Try It Yourself

- Break the ledger on purpose: post an entry set where debits and credits
  don't match, and confirm `ErrUnbalanced` rejects it — then check that
  **nothing** was written (query the account's balance before and after)
- Add a `Status: Pending` state to `paymentgateway`'s deposits, only
  transitioning to settled once the webhook arrives — right now the demo
  doesn't expose an intermediate state to query, which a real API would need
- Add a fourth reconciliation check: verify that `sum(all wallet balances)
  + sum(all savings balances) - sum(all loan pool balances) == ` some
  expected platform-wide total — a good exercise in extending
  `CheckLedgerIntegrity`'s spirit to a specific business invariant instead
  of just the structural one
- Expose `Transfer`, `PaymentGateway`, and the others as real HTTP APIs
  (Module 15/16's patterns) or gRPC services (Module 20's patterns) instead
  of direct function calls — the ledger and every service underneath
  wouldn't need to change at all, exactly like Module 19's ports-and-
  adapters architecture promises
