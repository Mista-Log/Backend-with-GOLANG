# Go for Beginners — Module 15: HTTP

## Contents

1. **[15-http.md](./15-http.md)** — `net/http` fundamentals (the
   goroutine-per-request model, tying directly back to Module 12), Request
   and Response (including the "headers before body" ordering rule),
   middleware as composed higher-order functions, routing with Go 1.22+'s
   native `ServeMux` path parameters, cookies (with `HttpOnly`/`Secure`/
   `SameSite` explained), headers and content negotiation, CORS (the full
   browser preflight sequence, diagrammed step by step), and streaming via
   Server-Sent Events with `http.Flusher` and context-aware cancellation.
   Diagrams throughout every section.

2. **[rest-server/](./rest-server)** — A complete, runnable task-management
   REST API exercising every section of the guide together: a 5-layer
   middleware chain (recover → log → request-ID → CORS → auth), Go 1.22+
   path-parameter routing, cookie-based session login, method-based (not
   path-based) auth enforcement, and a live SSE streaming endpoint. The
   README is deliberately diagram-heavy — a full request-lifecycle diagram,
   a routing table, sequence diagrams for the login→create→read flow and
   the CORS preflight handshake, and two case studies on real debugging
   traps (auth ordering breaking CORS, and why auth checks the HTTP method
   rather than a path list).

## Suggested Order

```
HTTP guide ──▶ REST Server
```

This module has one project, deliberately comprehensive rather than split
across several smaller ones — HTTP's pieces (routing, middleware, cookies,
CORS) only really make sense assembled together into one real request
lifecycle, which is exactly what the REST Server's main.go and README walk
through end to end.

## Quick Reference: Running and Exploring the Server

```bash
cd rest-server
go run .
# in another terminal — see the README for the full walkthrough:
curl http://localhost:8080/tasks
curl -i -c cookies.txt -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" -d '{"username":"ada","password":"x"}'
curl -i -b cookies.txt -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" -d '{"title":"Buy milk"}'
curl -N http://localhost:8080/tasks/stream   # -N: don't buffer, watch it stream live
```

*Note: this module builds on Modules 00–13 — start there first if you
haven't already, especially Module 06 (interfaces — a `Handler` is one),
Module 12 (concurrency — every request is its own goroutine), and Module 13
(context — streaming cancellation relies on it directly). Path-parameter
routing (`r.PathValue`, method-specific `ServeMux` patterns) requires Go
1.22+.*
