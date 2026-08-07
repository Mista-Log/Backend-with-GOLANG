# Project 1 — Payment Gateway

```bash
cd payment-gateway
go run main.go
```

This one's a demo script rather than an interactive menu — it walks through
all three payment types in a row so the differences (especially around
refunds) are visible side by side in one run.

## What's Demonstrated Here

- **`PaymentProcessor` interface** — `Gateway` only ever talks to this one
  method (`Process`), and never needs a `switch` on "is this a card, PayPal,
  or crypto?" anywhere. Adding a fourth payment method later means writing a
  new type with a `Process` method — zero changes to `Gateway` itself.
- **Embedding for composition** — every processor embeds `Logger`, so
  `c.Log(...)` works on all three without any of them writing their own
  logging code.
- **A *smaller*, separate `Refundable` interface** — deliberately **not**
  merged into `PaymentProcessor`, because not every payment method can
  support refunds (crypto, in this model, can't). This is the "small,
  composable interfaces" idiom from the guide, applied to a real design
  decision instead of just `Reader`/`Writer`.
- **Type assertion for optional behavior** — `TryRefund` checks
  `g.processor.(Refundable)` at runtime. `Gateway` never needed to know in
  advance which processors support refunds; it finds out by asking.
- **`any` for freeform metadata** — `Receipt.Metadata` can hold genuinely
  different keys depending on which processor produced it, without needing
  a different `Receipt` type per payment method.
- **Reflection for a generic receipt printer** — `printReceipt` walks
  `Receipt`'s fields via `reflect`, so it keeps working unmodified even if
  you add new fields to `Receipt` later.

```
┌──────────────────────────────────────────────────────────┐
│              type PaymentProcessor interface {                  │
│                  Process(amount float64) (string, error)           │
│              }                                                        │
│                                                                          │
│   CreditCardProcessor ──▶ satisfies PaymentProcessor AND Refundable       │
│   PayPalProcessor     ──▶ satisfies PaymentProcessor AND Refundable          │
│   CryptoProcessor     ──▶ satisfies PaymentProcessor ONLY                       │
│                                                                                       │
│   Gateway only ever holds a PaymentProcessor — it discovers Refundable                  │
│   support (or its absence) at runtime, via a type assertion, only when                    │
│   TryRefund is actually called.                                                              │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `Refundable` Is a Separate Interface, Not a Method on `PaymentProcessor`

The tempting shortcut would be to just add `Refund(...)` directly to
`PaymentProcessor`, and have `CryptoProcessor.Refund` return an
"unsupported" error. That technically works, but it's a worse design for two
reasons: first, it forces **every future payment method** to write a
`Refund` method even when the honest answer is "this can never happen" —
dead code that exists purely to satisfy an interface. Second, and more
importantly, it hides the capability difference from the *type system*:
callers can no longer tell, just by looking at what a processor implements,
whether refunds are even a supported concept for it. Keeping `Refundable`
separate means `TryRefund`'s type assertion is checking something real and
meaningful — "does this specific processor actually support this?" — rather
than always succeeding and returning a runtime error every single time for
processors that never will.

## Try It Yourself
- Add a fourth processor (e.g., `BankTransferProcessor`) that satisfies
  `PaymentProcessor` but not `Refundable` — confirm `Gateway` needs zero
  changes to accept it
- Add a `Recurring` interface (`Schedule(intervalDays int) error`) that only
  `CreditCardProcessor` and `PayPalProcessor` satisfy, and a
  `TrySchedule` method on `Gateway` mirroring `TryRefund`'s pattern
- Change `printReceipt` to also print each processor's `Prefix` field by
  reflecting into the embedded `Logger` — a good exercise in `reflect`
  handling nested/embedded struct fields
