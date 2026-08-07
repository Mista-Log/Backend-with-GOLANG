# Project 2 — Notification Service

```bash
cd notification-service
go run main.go
```

## What's Demonstrated Here

- **`MultiNotifier` satisfies `Notifier` by BEING one** — it has a `Send`
  method, so anywhere a single `Notifier` is expected, a `MultiNotifier`
  (holding any number of other `Notifier`s) works too. This is the
  **Composite** design pattern, and Go's implicit interface satisfaction
  makes it almost free to write — no special "composite" base class needed.
- **Interface embedding (`ValidatedNotifier`)** — composed from `Notifier`
  and `Validator`, mirroring the guide's `ReadWriter` example directly.
- **A channel-by-channel type assertion inside `MultiNotifier.Send`** —
  `ch.(Validator)` checks, per channel, whether THIS particular one happens
  to support validation. `EmailNotifier` and `SMSNotifier` do; `PushNotifier`
  doesn't — and `MultiNotifier` handles that mixed situation gracefully,
  without ever needing a type switch over "is this email, SMS, or push?"
- **A type SWITCH (`describe`)**, used specifically where a type
  ASSERTION wouldn't fit — `describe` needs to handle *every* possible
  concrete type differently (for a log label), not just check for one.
  Compare this against `MultiNotifier.Send`'s single-type assertion right
  above it — same underlying interface value, two different tools depending
  on whether you're checking for ONE type or branching across SEVERAL.
- **Nesting a `MultiNotifier` inside another `MultiNotifier`** — since a
  `MultiNotifier` is just another `Notifier` as far as the type system is
  concerned, this needs no special support at all; it falls out of the
  design for free.

```
┌──────────────────────────────────────────────────────────┐
│   everything := MultiNotifier{                                  │
│       channels: [                                                  │
│           EmailNotifier{...},                                        │
│           MultiNotifier{            ◀── a Notifier holding Notifiers,  │
│               channels: [                inside a Notifier holding      │
│                   SMSNotifier{...},        Notifiers — arbitrarily         │
│                   PushNotifier{...},        deep nesting, no extra code      │
│               ]                              required for it to work           │
│           },                                                                      │
│       ]                                                                              │
│   }                                                                                     │
│                                                                                            │
│   everything.Send("...")  fans out through EVERY level automatically                        │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Type Assertion vs. Type Switch, Side by Side

This project deliberately uses both, right next to each other, so the
distinction is concrete instead of abstract:

```go
// TYPE ASSERTION — "does this ONE specific type apply here?"
if v, ok := ch.(Validator); ok {
	// only branches on ONE question: yes or no
}

// TYPE SWITCH — "which of SEVERAL known types is this?"
switch v := n.(type) {
case EmailNotifier: ...
case SMSNotifier:   ...
case PushNotifier:  ...
default:            ...
}
```

Reach for a type assertion when you have one specific capability you're
checking for (exactly the `Refundable` pattern from Payment Gateway). Reach
for a type switch when you genuinely need different behavior for several
different concrete types and there's no shared method you could add to an
interface instead — though it's worth asking, when you find yourself writing
a big type switch, whether an interface method would actually be a cleaner
fit. (Here, `describe` is a reasonable type switch use: it's purely for
logging, and adding a `Describe()` method to every `Notifier` just for log
output would be more code than it's worth.)

## Try It Yourself
- Add a `SlackNotifier` and confirm it slots into `MultiNotifier` (and even
  nested `MultiNotifier`s) with zero changes to existing code
- Change `MultiNotifier.Send` to stop at the first failure (fail-fast)
  instead of collecting all failures — compare the trade-offs
- Add a `Priority` interface (`Priority() int`) that only some channels
  satisfy, and sort `MultiNotifier`'s channels by priority before sending,
  using a type assertion per channel to decide each one's priority (default
  to some baseline for channels that don't implement it)
