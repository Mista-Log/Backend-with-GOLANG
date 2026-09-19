# Money must be stored as integer minor units — float64 loses cents in ways that compound silently

**Source**: Module 21's fintech backend + Stripe/Monzo API docs
**Why I read this**: Wanted to know exactly *how badly* floats fail for
money, rather than just accepting "don't use floats" as folklore.

## The claim, in my words

`float64` cannot represent most decimal fractions exactly. 0.1 + 0.2 is
famously not 0.3. In a ledger, those tiny errors accumulate across
thousands of transactions until the books genuinely don't balance — and
a ledger that doesn't balance is a ledger you can't trust at all.

Store cents (or the smallest unit for the currency) as `int64`. Format
for display only at the very edge, never in the domain logic.

## Concrete example

```go
// The failure, demonstrated
a := 0.1
b := 0.2
fmt.Println(a+b == 0.3)        // false
fmt.Printf("%.20f\n", a+b)     // 0.30000000000000004441

// Accumulating 10,000 small transactions
var floatTotal float64
for i := 0; i < 10000; i++ { floatTotal += 0.01 }
fmt.Printf("%.10f\n", floatTotal)   // 100.0000000000 — but NOT exactly 100

var intTotal int64
for i := 0; i < 10000; i++ { intTotal += 1 }   // 1 cent each
fmt.Println(intTotal)                           // 10000 cents = exactly $100.00
```

**What actually happened when I ran it**: the float total printed
`99.9999999999906` at higher precision — off by a measurable amount after
only 10,000 operations. A real payment system does that many in minutes.

## The int64 approach in practice

```go
type Money int64  // always minor units — cents, kobo, etc.

func (m Money) String() string { return fmt.Sprintf("$%.2f", float64(m)/100) }
// float64 appears ONLY here, at the display boundary, never in arithmetic
```

## When this applies / doesn't

**Always applies** to balances, transaction amounts, and anything that
must reconcile exactly.

**Doesn't apply** to genuinely approximate quantities — an interest *rate*
(5.25%) or an exchange rate is legitimately fractional. Those stay
float64 (or `decimal`), but the *result* of applying them gets rounded
back to integer minor units immediately, with the rounding rule chosen
deliberately and documented.

**The remaining hard part**: rounding. Applying 3.7% interest to 1,247
cents gives 46.139 cents. Round half-up? Banker's rounding? Whatever you
choose, choose it explicitly and apply it consistently — inconsistent
rounding is its own reconciliation bug.

## Links

- Module 21 (the ledger, which uses int64 minor units throughout)
- Related: `notes/09-postmortems/` — rounding bugs appear in real incidents
