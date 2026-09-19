# Postmortems from Large Engineering Teams

The most under-read technical writing available. Real failures, described
honestly, by teams with resources you probably don't have — which means
the failure modes are *structural*, not "they were careless."

Use `templates/postmortem-template.md` for these; incident reports have a
different shape than concept notes.

## Where to find good ones

- AWS post-event summaries (`aws.amazon.com/premiumsupport/technology/pes/`)
- Cloudflare's blog — consistently detailed and candid
- GitHub's availability reports
- GitLab's 2017 database incident writeup (a classic)
- `danluu.com/postmortems` — a large curated collection
- Google SRE Book, ch. 15 (how to write one, not just read one)

## What to extract from each

Not "what broke" — that's rarely reusable. Extract the **failure chain**
and **which safeguard was missing or mis-tuned**. Those transfer.

## Recurring themes worth tracking across many postmortems

- Retry storms with no jitter amplifying a small failure
- Health checks testing the wrong thing (Module 19's liveness/readiness)
- Cache stampedes after an eviction
- Config changes deployed faster than code changes, with less review
- Cascading failure from a missing circuit breaker (Module 26)

## Notes in this folder

- `retry-storms-need-jitter.md`

## Related modules

Module 26 (circuit breaker, backpressure), Module 19 (health checks), Module 08 (retry)
