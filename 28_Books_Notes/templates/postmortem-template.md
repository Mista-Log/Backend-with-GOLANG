# <Company> — <one-line description of the outage>

**Source**: <link to the public postmortem> | **Date of incident**: <date>
**Duration / blast radius**: <how long, how many users/systems>

## What broke, in one paragraph

<Plain language. No jargon from the original unless you can define it.>

## The failure chain

```
<trigger> -> <what it caused> -> <what THAT caused> -> <user-visible impact>
```

<Most serious outages are chains, not single failures. Mapping the chain
is the whole exercise — a single "root cause" is usually a simplification
that hides the more interesting story.>

## Which safeguard was missing, or present-but-insufficient

<Be specific: was there no circuit breaker? One with a threshold set too
high? A retry storm with no jitter? A health check testing the wrong thing?>

## The pattern I already know that applies here

<Connect it to something you've built. Module 26's circuit breaker,
Module 21's idempotency keys, Module 19's liveness-vs-readiness
distinction, Module 26's backpressure.>

## What I'd check in my own systems because of this

- [ ] <a specific, actionable check — not "be more careful">
- [ ] <another>

## Links

- Related postmortem: <path>
- Related note: <path>
