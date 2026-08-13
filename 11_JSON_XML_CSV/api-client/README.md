# Project 1 — API Client

```bash
cd api-client
go run main.go
```

This starts a local HTTP server via `httptest` (a random local port, closed
automatically when the program exits) and immediately talks to it — nothing
external, no real network dependency, fully deterministic.

## What's Demonstrated Here

- **The same `User` struct, tagged for both JSON and XML** — `json:"..."`
  and `xml:"..."` tags coexist on the same fields, and `/users` vs.
  `/users.xml` serve the identical underlying data through two different
  encoders.
- **Streaming decode from an HTTP response body** — `fetchUsersJSON` and
  `fetchUsersXML` both call `Decode` directly on `resp.Body` (an
  `io.Reader`), never buffering the whole response into a `[]byte` first —
  Module 10's streaming habit, applied to network I/O instead of files.
- **Encoding a request body** — `createUser` builds a JSON payload with
  `json.Marshal` and sends it as a POST body, then decodes the JSON
  response — encoding and decoding on two ends of the same request.
- **Custom `UnmarshalJSON` for inconsistent upstream data** — the seed data
  deliberately uses three different timestamp formats (simulating "this API
  aggregates data from several other systems that don't agree on a date
  format"), and `FlexibleTime.UnmarshalJSON` tries each known layout in turn.

```
┌──────────────────────────────────────────────────────────┐
│   "2026-01-15T09:30:00Z"     ──▶  tries RFC3339 first  ──▶  MATCH        │
│   "2026-02-20T14:00:00"       ──▶  tries RFC3339 (fails)                     │
│                                     tries "2006-01-02T15:04:05"  ──▶ MATCH       │
│   "2026-03-05"                 ──▶  tries RFC3339 (fails)                          │
│                                      tries "2006-01-02T15:04:05" (fails)               │
│                                      tries "2006-01-02"  ──▶ MATCH                        │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `FlexibleTime` Is Deliberately Asymmetric

The guide's Custom Unmarshal section warns that `MarshalJSON` and
`UnmarshalJSON` should generally stay symmetric, so round-tripping a value
doesn't silently corrupt it — and the CSV Importer project's `Money` type
follows that rule strictly. `FlexibleTime` **intentionally breaks it**:
`UnmarshalJSON` is overridden to accept three different input formats, but
`MarshalJSON` is left as whatever `time.Time`'s own (promoted, via
embedding) method already does — always RFC3339.

This is a deliberate application of **Postel's Law** ("be liberal in what
you accept, conservative in what you send") — genuinely appropriate here,
specifically *because* `FlexibleTime` sits at an **ingestion boundary**: its
whole purpose is smoothing over messy, inconsistent data coming *in* from
elsewhere, while this program's own *outgoing* data (the POST response, or
anything this program itself produces) stays in one clean, predictable
format. The symmetry warning from the guide still applies to `Money` because
`Money` isn't absorbing external inconsistency — it's this program's own
canonical representation of currency, so `Marshal` and `Unmarshal` genuinely
ought to agree.

## Try It Yourself
- Add a fourth seed timestamp in a format `FlexibleTime` doesn't yet handle,
  confirm you get a clear decode error, then add that layout to
  `acceptedLayouts` and confirm it now parses
- Add `DisallowUnknownFields()` to `fetchUsersJSON`'s decoder and send a
  response with an extra, unexpected key — observe it now becomes a hard
  error instead of being silently ignored
- Extend the XML side to also carry `CreatedAt`, by giving `FlexibleTime`
  its own `UnmarshalXML`/`MarshalXML` methods (the XML equivalents of the
  JSON ones) — a good exercise in seeing how similar `encoding/xml`'s custom
  hooks are to `encoding/json`'s
