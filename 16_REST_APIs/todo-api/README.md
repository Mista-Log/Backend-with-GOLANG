# Project 1 — Todo API (Gin)

A CRUD API built with **Gin**, covering validation, pagination, and
Swagger/OpenAPI annotations.

## Setup

This project depends on Gin, an external package — unlike every previous
project in this course, it needs a one-time internet-connected setup step:

```bash
cd todo-api
go mod tidy    # fetches Gin and generates go.sum — needs internet access
go run .
```

```
┌──────────────────────────────────────────────────────────┐
│   go mod tidy                                                    │
│        │                                                            │
│        ▼                                                              │
│   reads go.mod's `require github.com/gin-gonic/gin v1.10.0`             │
│        │                                                                   │
│        ▼                                                                      │
│   downloads Gin (and its own dependencies, like go-playground/validator)         │
│   from the Go module proxy, writes go.sum with checksums (Module 09)                │
│        │                                                                                │
│        ▼                                                                                   │
│   go run . now compiles successfully                                                          │
└──────────────────────────────────────────────────────────┘
```

## Full curl Walkthrough

### Create a few todos

```bash
curl -X POST http://localhost:8080/todos -d '{"title": "Buy milk"}'
curl -X POST http://localhost:8080/todos -d '{"title": ""}'   # deliberately invalid
```
```
┌──────────────────────────────────────────────────────────┐
│   POST /todos   {"title": "Buy milk"}                                │
│        │                                                                │
│        ▼                                                                  │
│   c.ShouldBindJSON  →  binding:"required,min=1,max=200" PASSES              │
│        │                                                                       │
│        ▼                                                                         │
│   201 Created   {"id":4,"title":"Buy milk","done":false,"createdAt":"..."}          │
│                                                                                          │
│   POST /todos   {"title": ""}                                                             │
│        │                                                                                     │
│        ▼                                                                                        │
│   c.ShouldBindJSON  →  binding:"required" FAILS (empty string)                                    │
│        │                                                                                              │
│        ▼                                                                                                 │
│   400 Bad Request   {"error": "Key: 'CreateTodoRequest.Title' Error:..."}                                    │
│                                                                                                                   │
│   createTodo's own logic NEVER RUNS for the second request — validation                                            │
│   rejected it before the handler body's first line.                                                                   │
└──────────────────────────────────────────────────────────┘
```

### Pagination

```bash
for i in $(seq 1 25); do curl -s -X POST http://localhost:8080/todos -d "{\"title\": \"Task $i\"}" > /dev/null; done

curl "http://localhost:8080/todos?page=1&limit=10"
curl "http://localhost:8080/todos?page=3&limit=10"
```
```
┌──────────────────────────────────────────────────────────┐
│   28 total todos, limit=10                                          │
│                                                                          │
│   page=1  →  {"data":[...10 items...],"page":1,"limit":10,                │
│                "total":28,"totalPages":3}                                    │
│   page=2  →  items 11-20                                                       │
│   page=3  →  items 21-28  (only 8 items — the last, PARTIAL page)                 │
│   page=4  →  {"data":[],"page":4,...}   (empty — past the end, not an error)         │
└──────────────────────────────────────────────────────────┘
```

### PUT vs. PATCH — the actual difference, demonstrated

```bash
# PUT replaces EVERYTHING — omitting "done" resets it to false
curl -X PUT http://localhost:8080/todos/1 -d '{"title": "Buy oat milk"}'
# {"id":1,"title":"Buy oat milk","done":false,...}  ◀── done was already true? Now it's false.

# PATCH touches ONLY what you send
curl -X PATCH http://localhost:8080/todos/1 -d '{"done": true}'
# {"id":1,"title":"Buy oat milk","done":true,...}  ◀── title is UNTOUCHED
```
```
┌──────────────────────────────────────────────────────────┐
│   PUT  {"title": "Buy oat milk"}                                     │
│        → Replace(id, "Buy oat milk", false)  ◀── Done NOT in the JSON,   │
│                                                    so Go's zero value        │
│                                                    (false) is what              │
│                                                    UpdateTodoRequest.Done         │
│                                                    ends up holding — and            │
│                                                    Replace WRITES it,                 │
│                                                    resetting Done                        │
│                                                                                             │
│   PATCH  {"done": true}                                                                       │
│         → PatchDone(id, true)  ◀── only ONE field is ever touched,                              │
│                                     Title is left completely alone                                 │
└──────────────────────────────────────────────────────────┘
```

### Trying to PATCH without the required field

```bash
curl -X PATCH http://localhost:8080/todos/1 -d '{}'
# 400 Bad Request — "done" is binding:"required" on a *bool
```

## Case Study: Why `PatchTodoRequest.Done` Is a `*bool`, Not a `bool`

```go
type PatchTodoRequest struct {
	Done *bool `json:"done" binding:"required"`
}
```

A plain `bool` field's zero value is `false` — completely indistinguishable
from "the client explicitly sent `false`." With a `*bool`, there are three
possible states after binding: `nil` (the field was omitted entirely — and
`binding:"required"` rejects this case before the handler even runs),
`&false` (explicitly set to false), and `&true` (explicitly set to true).
This is the exact same "does the zero value mean something, or mean
nothing was said" question Module 04's zero-values section raised — a
pointer is the standard Go idiom for "optional, with a real absent state,"
used here specifically so `PATCH {"done": false}` behaves correctly instead
of being silently indistinguishable from an empty PATCH body.

## Swagger / OpenAPI

Every handler in `handlers.go` has a `@`-annotated godoc comment (see the
guide's Swagger section). To generate and serve interactive docs:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init
go get github.com/swaggo/gin-swagger github.com/swaggo/files
```
Then add to `main.go`:
```go
import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "todoapi/docs" // generated by `swag init`
)
// ...
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```
Then visit `http://localhost:8080/swagger/index.html` for a fully
interactive, browsable API explorer generated entirely from the comments
already sitting above each handler — no separate documentation to maintain.

## Try It Yourself
- Add a `priority` field (`low`/`medium`/`high`) with a `binding:"oneof=low medium high"`
  validation tag, and confirm an invalid value is rejected automatically
- Add cursor-based pagination as an alternative `?cursor=` mode, per the
  guide's comparison — think through what the "cursor" value should encode
  for this todo list specifically
- Run `swag init` for real and get the interactive docs page loading — it's
  a genuinely satisfying result for how little extra code it needs
