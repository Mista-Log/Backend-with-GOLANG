# 21. FinTech Backend

*This is where you become industry-ready.*

Every module before this taught general-purpose backend engineering. This
one is domain-specific: the patterns that show up in every real payments,
banking, or wallet system, regardless of which language or framework
implements them. Get the ideas in this module right, and the code follows
naturally — get them wrong, and no amount of clever Go saves you from
losing track of someone's money.

---

## Double Entry Accounting

The foundational rule underneath every real financial system: **every
transaction affects at least two accounts, and the total debits always
equal the total credits.** Money is never created or destroyed by a
transaction — it only ever *moves*, from a labeled source to a labeled
destination.

```
┌──────────────────────────────────────────────────────────┐
│              A single-entry mistake (DON'T do this)                 │
│                                                                          │
│   "Kemi's balance: +$50"    ◀── where did this $50 come FROM?              │
│                                    Nothing in the record says.                │
│                                                                                   │
│              The SAME event, double-entry                                          │
│                                                                                         │
│   DEBIT   Cash (asset account)              $50                                          │
│   CREDIT  Kemi's Wallet (liability account)   $50                                            │
│                                                                                                   │
│   Every dollar has a NAMED origin and a NAMED destination — the books                               │
│   ALWAYS balance (debits == credits), which makes errors, fraud, and                                   │
│   bugs structurally easier to detect: if the books ever DON'T balance,                                    │
│   something is definitively, provably wrong.                                                                 │
└──────────────────────────────────────────────────────────┘
```

**Debit and credit don't mean "subtract" and "add"** — which one increases
a balance depends on the account's **type**:

```
┌────────────────────────────────────────────────────┐
│   ASSET accounts (cash, receivables)      → DEBIT increases them          │
│   LIABILITY accounts (customer wallets)    → CREDIT increases them            │
│   EQUITY accounts                           → CREDIT increases them               │
│   REVENUE accounts                           → CREDIT increases them                  │
│   EXPENSE accounts                            → DEBIT increases them                     │
│                                                                                              │
│   A customer's wallet balance is a LIABILITY from the platform's own            │
│   point of view (money the platform OWES the customer) — this is why                │
│   crediting a customer's wallet is the correct direction for a deposit,                │
│   even though it "feels like" the number just went up either way.                          │
└────────────────────────────────────────────────────┘
```

This module's project implements this literally: every operation —
deposit, transfer, escrow, loan disbursement — posts a balanced set of
debit/credit entries, never a single "add $50" mutation.

---

## Ledger

The **ledger** is the append-only, immutable record of every entry ever
posted — the single source of truth every account balance is *derived
from*, not a separate thing balances are synced with.

```
┌──────────────────────────────────────────────────────────┐
│   Transaction #1: "Ada deposits $100"                                │
│      Entry: DEBIT  Cash              $100                              │
│      Entry: CREDIT Ada's Wallet       $100                                │
│                                                                               │
│   Transaction #2: "Ada sends $30 to Bob"                                       │
│      Entry: DEBIT  Ada's Wallet       $30                                        │
│      Entry: CREDIT Bob's Wallet        $30                                          │
│                                                                                          │
│   Ada's balance = SUM of every entry ever posted to Ada's Wallet account:                  │
│      +$100 (credit, transaction #1) - $30 (debit, transaction #2) = $70                       │
│                                                                                                    │
│   The balance is a QUERY over history, not a mutable field that could                               │
│   drift out of sync with what actually happened.                                                       │
└──────────────────────────────────────────────────────────┘
```

**Nothing in a real ledger is ever edited or deleted** — a mistake is
corrected with a new, offsetting **reversal** transaction, never by
changing history. This is the same principle as Module 17's migrations
(forward-only, numbered, never rewritten) and Git commits, applied to
money: the record of what happened is permanent; only the *current state*
(derived by replaying/summing it) changes.

---

## Wallet

A **wallet** is the customer-facing abstraction sitting on top of one or
more ledger accounts — "Kemi's balance is $70" is really "the sum of every
ledger entry posted to the liability account representing Kemi's wallet."

```
┌────────────────────────────────────────────────────┐
│    Customer-facing view:        Underlying reality:                       │
│                                                                                │
│    Kemi's Wallet: $70            Ledger account "wallet:kemi" (LIABILITY)        │
│                                    balance = sum of its posted entries               │
│                                                                                          │
│    A wallet with multiple currencies is really MULTIPLE ledger                            │
│    accounts (one per currency) presented together as one UI concept.                         │
└────────────────────────────────────────────────────┘
```

---

## Transactions

In this module, "transaction" means a **balanced group of ledger entries**
posted atomically — either the whole group is recorded, or none of it is
(Module 17's database transactions, applied at the business-logic level:
you never want half of a transfer to post).

```
┌──────────────────────────────────────────────────────────┐
│   PostTransaction([                                                  │
│       {Account: "wallet:ada", Direction: Debit,  Amount: 30},           │
│       {Account: "wallet:bob", Direction: Credit, Amount: 30},              │
│   ])                                                                          │
│        │                                                                        │
│        ▼                                                                          │
│   VALIDATE: sum(debits) == sum(credits) for EVERY currency involved                  │
│        │  NO  → reject the WHOLE transaction, nothing is posted                        │
│        │  YES → append every entry, atomically, under one lock                            │
│        ▼                                                                                     │
│   the transaction is now PART OF HISTORY, permanently                                          │
└──────────────────────────────────────────────────────────┘
```

---
## Settlement

Many payment systems don't move real money on every single transaction —
they record obligations continuously, then **settle** them in a batch
(hourly, daily) by making one net real-money movement per counterparty.

```
┌──────────────────────────────────────────────────────────┐
│   Throughout the day: 500 individual card payments recorded             │
│   in the ledger against "Pending Settlement" accounts                       │
│                                                                                  │
│   End of day, ONE batch job:                                                       │
│      sum everything owed TO each merchant                                            │
│      make ONE real bank transfer per merchant, for their net total                      │
│      post a "Settled" transaction moving funds from Pending → Settled                      │
│                                                                                                 │
│   Why batch instead of real-time: real bank transfers/card network                                │
│   settlements have real per-transaction costs and processing windows —                              │
│   netting many small obligations into one settlement is dramatically                                   │
│   cheaper and simpler to reconcile than moving real money 500 times.                                       │
└──────────────────────────────────────────────────────────┘
```

---

## Reconciliation

Continuously verifying that the ledger's internal story **matches
reality** — an external bank statement, a card network's settlement
report, or even just the ledger's own internal invariant (do debits still
equal credits, system-wide?).

```
┌──────────────────────────────────────────────────────────┐
│   Internal reconciliation:                                            │
│      sum of ALL debit entries, ever  ==  sum of ALL credit entries?         │
│      NO → something is structurally broken RIGHT NOW — alert immediately      │
│                                                                                    │
│   External reconciliation:                                                          │
│      ledger says "$14,203 in virtual account deposits today"                            │
│      the REAL bank's statement says "$14,150 received today"                               │
│      $53 discrepancy → investigate: a delayed deposit? A fee that                              │
│      wasn't recorded? A fraudulent entry?                                                        │
│                                                                                                       │
│   Reconciliation is what catches the bugs and fraud that PASSED every                                   │
│   other check — it's the last line of defense, and real fintech systems                                    │
│   run it continuously, not just occasionally.                                                                  │
└──────────────────────────────────────────────────────────┘
```

---

## Payment Gateway

The component that talks to the **outside world** of real money movement —
card networks, bank rails, mobile money — translating your internal ledger
operations into real-world transfers, and real-world events (a card
charge succeeding) back into ledger entries.

```
┌──────────────────────────────────────────────────────────┐
│   Your system          Payment Gateway            The real world             │
│                                                                                    │
│   "charge $50 to        ──▶  talks to the card       ──▶  Visa/Mastercard/            │
│    this card"                 network's real API             the customer's bank        │
│                                                                                                │
│   (waits...)             ◀── async result: success/       ◀── (takes real time)             │
│                                failure, often via a                                              │
│                                WEBHOOK (below), not an                                              │
│                                immediate response                                                     │
│        │                                                                                                 │
│        ▼                                                                                                    │
│   ONLY on confirmed success does your system POST the ledger transaction                                       │
│   crediting the customer — never optimistically before confirmation                                               │
└──────────────────────────────────────────────────────────┘
```

---

## Escrow

Funds held by a **neutral third party** (your platform) until agreed
conditions are met — common in marketplaces (hold the buyer's payment
until the item ships), rentals, and any two-party transaction needing
trust neither side has to extend to the other directly.

```
┌──────────────────────────────────────────────────────────┐
│   Buyer pays $200  ──▶  DEBIT Buyer's Wallet $200                    │
│                          CREDIT Escrow Account $200 (tagged: escrow #42) │
│                                                                              │
│   ... time passes, seller ships the item ...                                  │
│                                                                                    │
│   Buyer confirms receipt  ──▶  DEBIT Escrow Account $200 (escrow #42)                │
│                                  CREDIT Seller's Wallet $200                            │
│                                                                                             │
│   (or, if the deal falls through)                                                             │
│                                                                                                    │
│   Refund  ──▶  DEBIT Escrow Account $200 (escrow #42)                                                │
│                 CREDIT Buyer's Wallet $200                                                             │
│                                                                                                             │
│   At every step, the money is FULLY ACCOUNTED FOR in a real ledger                                            │
│   account — "in escrow" isn't a status flag on a wallet, it's actually,                                          │
│   physically (in ledger terms) sitting in a different account.                                                      │
└──────────────────────────────────────────────────────────┘
```

---

## Webhook

An **inbound HTTP callback** — instead of your system polling "is the
payment done yet?" repeatedly, the payment gateway (or bank, or card
network) calls **your** server the moment something happens.

```
┌──────────────────────────────────────────────────────────┐
│   Your server:  POST /webhooks/payment-gateway                     │
│                                                                          │
│   {                                                                        │
│     "eventType": "payment.succeeded",                                        │
│     "paymentID": "pay_8f2a...",                                                 │
│     "idempotencyKey": "evt_9c31..."                                               │
│   }                                                                                  │
│                                                                                          │
│   Your handler MUST:                                                                        │
│     1. verify the request is genuinely FROM the gateway (a signature                           │
│         check, typically — never trust an unauthenticated webhook)                                │
│     2. check the idempotency key (below) BEFORE processing anything                                  │
│     3. respond 200 QUICKLY — most gateways RETRY on timeout/non-200,                                    │
│         so slow processing inside the handler causes duplicate deliveries                                  │
│     4. do the actual (possibly slower) work asynchronously if needed                                          │
└──────────────────────────────────────────────────────────┘
```

---

## Idempotency

The property that performing the same operation multiple times has the
**same effect** as performing it once — essential wherever a network call
might be retried (a timed-out request that actually succeeded, a
redelivered webhook, a user double-clicking "Pay").

```go
func (g *Gateway) HandleWebhook(event WebhookEvent) error {
	if g.alreadyProcessed(event.IdempotencyKey) { // check FIRST, always
		return nil // already handled — return success WITHOUT reprocessing
	}
	// ... post the ledger transaction ...
	g.markProcessed(event.IdempotencyKey)
	return nil
}
```

```
┌──────────────────────────────────────────────────────────┐
│   Without idempotency:                                                │
│     webhook delivered TWICE (a common, expected occurrence — most         │
│     gateways explicitly document "at-least-once" webhook delivery,          │
│     see Module 20) → customer's wallet credited TWICE → real money            │
│     created out of nowhere, a genuine financial bug                              │
│                                                                                       │
│   With idempotency (an idempotency key checked BEFORE any mutation):                    │
│     second delivery is recognized as a duplicate and safely ignored                        │
└──────────────────────────────────────────────────────────┘
```

This is Module 20's idempotent-consumer concept, but in fintech it isn't
an optimization — it's the difference between a correct system and one
that occasionally, unpredictably, gives people free money or double-charges
them.

---
## Fraud Detection

Automated checks looking for patterns associated with fraudulent activity
— never perfect, always a trade-off between catching real fraud and
inconveniencing real customers with false positives.

```go
func (c *ComplianceService) CheckTransaction(customerID string, amount float64) Decision {
	if amount > c.velocityThreshold(customerID) {
		return Decision{Action: Flag, Reason: "unusual transaction velocity"}
	}
	if amount > largeTransactionThreshold {
		return Decision{Action: Review, Reason: "large transaction requires manual review"}
	}
	return Decision{Action: Allow}
}
```

```
┌────────────────────────────────────────────────────┐
│   Common signals real systems check (far beyond this module's         │
│   simplified example): transaction VELOCITY (too many, too fast),         │
│   unusual AMOUNT relative to history, geographic/device anomalies,           │
│   known bad actor lists, behavioral pattern deviation — usually                 │
│   combined via a scored model, not a single hardcoded rule                        │
└────────────────────────────────────────────────────┘
```

---

## KYC (Know Your Customer)

Verifying a customer's real-world identity **before** letting them
transact meaningfully — legally required for regulated financial services
in most jurisdictions, and the practical foundation fraud/AML checks build
on (you can't flag "unusual behavior for this customer" if you don't
reliably know who the customer *is*).

```
┌──────────────────────────────────────────────────────────┐
│   Unverified account:  can view balance, maybe receive small         │
│                          amounts — but LOW transaction limits,             │
│                          often blocked from withdrawing at all                │
│                                                                                    │
│   KYC-verified account (ID document + often a selfie/liveness check,                 │
│                           verified against the document): full limits,                  │
│                           full feature access                                              │
└──────────────────────────────────────────────────────────┘
```

---

## AML (Anti-Money Laundering)

Regulatory obligations to detect and report suspicious activity that could
indicate money laundering — structuring transactions to avoid reporting
thresholds, transactions with sanctioned entities/countries, and unusual
patterns inconsistent with a customer's known profile.

```
┌────────────────────────────────────────────────────┐
│   A transaction just under a reporting threshold, repeated MANY               │
│   times in a short window ("structuring") is a classic AML red flag,             │
│   precisely BECAUSE it looks like someone deliberately avoiding                     │
│   triggering a single large-transaction report                                        │
└────────────────────────────────────────────────────┘
```

**KYC and AML are legal/regulatory domains, not just engineering
problems** — real implementations integrate with licensed identity
verification and sanctions-screening providers rather than hand-rolling
checks; this module's project implements the *pattern* (a compliance check
in the transaction path) at a level appropriate for learning, not a
production-legal implementation.

---

## PCI DSS

**Payment Card Industry Data Security Standard** — the security
requirements any system touching card data must meet. The single most
important practical takeaway for a Go backend engineer:

```
┌──────────────────────────────────────────────────────────┐
│   NEVER store, log, or even touch raw card numbers (PANs) or CVVs         │
│   in your own systems unless you have gone through FULL PCI DSS               │
│   certification (expensive, ongoing, and usually not worth it for                │
│   most companies).                                                                  │
│                                                                                          │
│   Instead: use a PCI-compliant provider (Stripe, Adyen, Braintree...) that                 │
│   handles the raw card data entirely on THEIR infrastructure, and gives                       │
│   your system back a TOKEN — an opaque reference you can charge/refund                            │
│   without ever seeing the real card number at all.                                                    │
│                                                                                                            │
│   This is a real architectural decision, not just a compliance checkbox:                                     │
│   it's WHY almost no backend engineer, at almost no company, ever writes                                        │
│   code that touches a raw card number — and this module's project                                                 │
│   follows that same discipline, working only with wallets and ledger                                                 │
│   accounts, never simulating raw card storage at all.                                                                  │
└──────────────────────────────────────────────────────────┘
```

---

## Audit Logs

Two genuinely distinct kinds of "audit trail" a real fintech system
maintains, often confused for one thing:

```
┌──────────────────────────────────────────────────────────┐
│   THE LEDGER ITSELF is the FINANCIAL audit trail —                    │
│     "what happened to the MONEY, and when" — this module's                 │
│     append-only ledger design already IS this, structurally                   │
│                                                                                    │
│   A SEPARATE operational/security audit log answers                                 │
│     "who DID what, from where, when" — every API call, every                           │
│     admin action, every login — regardless of whether money                               │
│     moved at all (a failed login attempt, an admin viewing a                                 │
│     customer's data, a support agent issuing a refund)                                          │
└──────────────────────────────────────────────────────────┘
```

Real fintech systems need **both** — the ledger proves what happened to
the money; the operational audit log proves who was responsible for
triggering it, essential for both security investigations and regulatory
compliance (many regulations specifically require being able to answer
"who accessed this customer's financial data, and when").

---

## Multi Currency

Supporting more than one currency means every ledger account and entry
needs a **currency** field — and critically, **you can never post a
transaction whose debits and credits balance in DIFFERENT currencies
directly**; a currency conversion has to be an explicit, separate step.

```
┌──────────────────────────────────────────────────────────┐
│   ❌ WRONG — mixing currencies in one "balanced" transaction:            │
│      DEBIT  Ada's Wallet (USD)     $100                                     │
│      CREDIT Bob's Wallet (NGN)      150,000                                    │
│      (this "balances" numerically but conflates two different UNITS —              │
│       meaningless, and definitely not how real ledgers work)                          │
│                                                                                            │
│   ✅ RIGHT — an explicit FX conversion, through a suspense account:                          │
│      DEBIT  Ada's Wallet (USD)          $100                                                    │
│      CREDIT FX Suspense (USD)            $100                                                      │
│      ── separate transaction, using the current exchange rate ──                                      │
│      DEBIT  FX Suspense (NGN)             150,000                                                        │
│      CREDIT Bob's Wallet (NGN)             150,000                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## Exchange Rates

The conversion factor between currencies, which **changes constantly** in
reality — a real system fetches live rates from a market data provider;
this module's project uses a simple in-memory rate table standing in for
that external dependency, exactly like earlier modules stood in for
external APIs and OAuth providers.

```go
type ExchangeRateService struct {
	rates map[string]float64 // e.g. "USD:NGN" -> 1550.00
}

func (s *ExchangeRateService) Convert(amount float64, from, to string) (float64, error) {
	rate, ok := s.rates[from+":"+to]
	if !ok {
		return 0, fmt.Errorf("no exchange rate for %s to %s", from, to)
	}
	return amount * rate, nil
}
```

**Real systems record the exact rate used, on every conversion, forever**
— rates change by the minute, and a converted transaction from last
Tuesday needs to show *which* rate applied at that moment, both for
customer trust and for accounting/audit purposes.

---

Onto the project — one integrated fintech backend, built around a shared
double-entry ledger core, with every topic above implemented as a real
module sitting on top of it: Wallet, Transfer, Virtual Accounts, Payment
Gateway (with real webhooks and idempotency), Escrow, Savings, Loans,
Reconciliation, and Compliance (fraud/KYC/AML checks) — because in real
fintech engineering, these aren't nine separate apps; they're nine products
built on one ledger.
