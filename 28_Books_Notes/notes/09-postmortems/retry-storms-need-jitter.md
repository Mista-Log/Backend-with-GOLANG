# Synchronized retries turn one brief failure into a sustained outage — jitter is not optional

**Source**: AWS Architecture Blog "Exponential Backoff and Jitter" +
recurring theme across many public postmortems
**Why I read this**: Module 08's retry package has exponential backoff but
no jitter, and I wanted to know how much that actually matters.

## What breaks

A service blips for 2 seconds. Ten thousand clients all fail at once,
all wait exactly 1s, all retry at the *same instant*, overwhelming the
recovering service — which fails again. Now they all wait 2s, and retry
in unison again. The synchronization is self-sustaining.

## The failure chain

```
brief backend blip -> N clients fail simultaneously -> all back off by the
SAME amount -> all retry simultaneously -> backend overwhelmed at exactly
the wrong moment -> more failures -> the retry wave repeats, amplified
```

The critical insight: exponential backoff alone **doesn't help**, because
every client is backing off *in lockstep*. It spaces out the waves, but
each wave is just as concentrated.

## The fix

```go
// Without jitter — all clients retry at identical times
delay := baseDelay * time.Duration(1<<attempt)

// With full jitter — retries spread across the window
backoff := baseDelay * time.Duration(1<<attempt)
delay := time.Duration(rand.Int63n(int64(backoff)))
```

AWS's analysis found "full jitter" (random between 0 and the backoff
window) outperformed both no-jitter and partial-jitter approaches on
both total completion time and server load.

## The pattern I already know that applies here

Module 26's **circuit breaker** is the complementary fix: jitter
desynchronizes the retries, while the breaker stops them entirely once
it's clear the dependency is genuinely down rather than blipping. Neither
alone is sufficient — jittered retries against a truly-dead service still
generate sustained load; a breaker without jitter still produces a
thundering herd the moment it closes again.

## What I'd check in my own systems because of this

- [ ] Does every retry path have jitter, not just backoff?
- [ ] When a circuit breaker transitions Open -> Half-Open, does exactly
      one request go through, or do all waiting clients rush in at once?
- [ ] Do client SDKs we publish default to jittered retries, or leave it
      to callers who won't know to add it?

## Links

- Module 08 (the retry package — add jitter to it as an exercise)
- Module 26 (circuit breaker, backpressure)
