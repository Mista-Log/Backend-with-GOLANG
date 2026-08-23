# Project — REST Server

A complete, runnable REST API tying together every piece of Module 15:
routing, request/response handling, a chained middleware stack, cookie-based
session auth, CORS for browser access, custom headers, and a live streaming
endpoint — all built around one small task-management resource.

```bash
cd rest-server
go run .
# REST server listening on http://localhost:8080
```

This README leans heavily on diagrams throughout — HTTP is inherently about
things flowing between processes (browser ⇄ server, layer ⇄ layer), and
that's much easier to actually absorb visually than as prose alone.

---

## 1. The Whole Picture, First

Before diving into any one piece, here's how a single request moves through
this entire program, end to end:

```
┌────────────────────────────────────────────────────────────────┐
│                                                                        │
│   Browser / curl                                                        │
│        │                                                                   │
│        │ HTTP request over TCP                                              │
│        ▼                                                                      │
│  ┌──────────────────────────────────────────────────────┐                       │
│  │                    net/http server                          │                       │
│  │           (a new goroutine per request — Module 12)           │                       │
│  │                                                                  │                       │
│  │   ┌────────────────────────────────────────────┐                  │                       │
│  │   │  recoverMiddleware   (catches ANY panic below)    │                  │                       │
│  │   │  ┌────────────────────────────────────────┐  │                  │                       │
│  │   │  │  loggingMiddleware   (times everything      │  │                  │                       │
│  │   │  │                        inside, incl. auth)    │  │                  │                       │
│  │   │  │  ┌────────────────────────────────────┐  │  │                  │                       │
│  │   │  │  │  requestIDMiddleware  (tags w/ an ID)  │  │  │                  │                       │
│  │   │  │  │  ┌────────────────────────────────┐  │  │  │                  │                       │
│  │   │  │  │  │  corsMiddleware  (headers +         │  │  │  │                  │                       │
│  │   │  │  │  │                    preflight handling) │  │  │  │                  │                       │
│  │   │  │  │  │  ┌────────────────────────────┐  │  │  │  │                  │                       │
│  │   │  │  │  │  │  authMiddleware                │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  (checks session cookie on         │  │  │  │  │                  │                       │
│  │   │  │  │  │  │   POST/PUT/DELETE only)              │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  ┌────────────────────────┐  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │   ServeMux ROUTER            │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  (matches METHOD + PATH,         │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │   extracts {id}, dispatches)       │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  ┌────────────────────┐  │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  │  your handler func     │  │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  │  (reads store,            │  │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  │   writes JSON)              │  │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  │  └────────────────────┘  │  │  │  │  │  │                  │                       │
│  │   │  │  │  │  │  └────────────────────────┘  │  │  │  │  │                  │                       │
│  │   │  │  │  │  └────────────────────────────┘  │  │  │  │                  │                       │
│  │   │  │  │  └────────────────────────────────┘  │  │  │                  │                       │
│  │   │  │  └────────────────────────────────────┘  │  │                  │                       │
│  │   │  └────────────────────────────────────────┘  │                  │                       │
│  │   └────────────────────────────────────────────┘                  │                       │
│  └──────────────────────────────────────────────────────┘                       │
│        │                                                                   │
│        │ HTTP response, same connection                                     │
│        ▼                                                                      │
│   Browser / curl receives it                                                    │
│                                                                                     │
└────────────────────────────────────────────────────────────────┘
```

Execution goes **in** through every layer in the order listed in `main.go`'s
`chain(...)` call, reaches your handler at the center, then unwinds **back
out** in reverse — the exact same LIFO shape as `defer` (Module 02) and a
call stack (Module 05). `main.go`'s comments call out exactly which
middleware is outermost vs. innermost.

---

## 2. Routing Table

```
┌──────────────────────────────────────────────────────────┐
│   METHOD   PATH                 AUTH REQUIRED?    HANDLER                │
│  ─────────────────────────────────────────────────────────  │
│   POST     /login                  no               loginHandler            │
│   GET      /tasks                   no               listTasksHandler          │
│   GET      /tasks/stream             no               streamHandler (SSE)         │
│   GET      /tasks/{id}                no               getTaskHandler                │
│   POST     /tasks                      YES              createTaskHandler               │
│   PUT      /tasks/{id}                  YES              updateTaskHandler                 │
│   DELETE   /tasks/{id}                   YES              deleteTaskHandler                  │
└──────────────────────────────────────────────────────────┘
```

`{id}` is a **path parameter** — Go 1.22+'s `http.ServeMux` extracts it
automatically, retrieved in a handler with `r.PathValue("id")`. Notice
`GET /tasks/stream` is registered as its own exact route *above*
`GET /tasks/{id}` conceptually — `ServeMux` always prefers the more specific
match, so a literal `/tasks/stream` never gets swallowed by the `{id}`
wildcard pattern.

---

## 3. Full curl Walkthrough

### Reading tasks (no login needed)

```bash
curl http://localhost:8080/tasks
```
```
┌──────────────────────────────────────────────────┐
│  curl ──▶ GET /tasks ──▶ [recover→log→reqID→cors→auth(skips, it's GET)→router]│
│                                       │                                          │
│                                       ▼                                            │
│                              listTasksHandler                                        │
│                                       │                                                │
│                                       ▼                                                  │
│  curl ◀── 200 OK  [{"id":1,"title":"...","done":false}, ...]                                │
└──────────────────────────────────────────────────┘
```

### Trying to create a task WITHOUT logging in first

```bash
curl -i -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk"}'
```
```
┌──────────────────────────────────────────────────┐
│  curl ──▶ POST /tasks ──▶ [recover→log→reqID→cors→AUTH]                    │
│                                              │                                  │
│                                       no session cookie present!                   │
│                                              ▼                                        │
│  curl ◀── 401 Unauthorized  {"error":"login required"}                                  │
│                                                                                             │
│  Notice: the request NEVER reaches the router or createTaskHandler at all —                    │
│  authMiddleware stopped it before either ran.                                                     │
└──────────────────────────────────────────────────┘
```

### Logging in, then creating a task

```bash
# -c saves cookies to a file; -b sends them back on later requests
curl -i -c cookies.txt -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username": "ada", "password": "anything"}'

curl -i -b cookies.txt -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk"}'
```
```
┌──────────────────────────────────────────────────────────┐
│   Request 1: POST /login                                                │
│        server ──▶ Set-Cookie: session=<random token>; HttpOnly; ...        │
│        curl's -c flag SAVES this to cookies.txt                                │
│                                                                                    │
│   Request 2: POST /tasks   (curl's -b flag SENDS it back)                            │
│        curl ──▶  Cookie: session=<same token>                                            │
│        authMiddleware:  r.Cookie("session") ──▶ sessions.Valid(token) ──▶ true              │
│        request PROCEEDS to createTaskHandler                                                    │
│                                                                                                       │
│   curl ◀── 201 Created  {"id":3,"title":"Buy milk","done":false}                                        │
└──────────────────────────────────────────────────────────┘
```

### Updating and deleting

```bash
curl -i -b cookies.txt -X PUT http://localhost:8080/tasks/3 \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk", "done": true}'

curl -i -b cookies.txt -X DELETE http://localhost:8080/tasks/3
```

### Watching the response headers (Request-ID, CORS)

```bash
curl -i http://localhost:8080/tasks
```
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: http://localhost:3000     ◀── corsMiddleware
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
X-Request-Id: 7f3a9c21                                    ◀── requestIDMiddleware
Content-Type: application/json
...
```

### A CORS preflight, by hand

```bash
curl -i -X OPTIONS http://localhost:8080/tasks \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type"
```
```
┌──────────────────────────────────────────────────────────┐
│   This is EXACTLY what a browser sends automatically, behind the        │
│   scenes, before letting your frontend JavaScript's real POST fetch()      │
│   call go out — see the guide's Section 8 diagram for the full browser        │
│   sequence this simulates.                                                       │
│                                                                                       │
│   curl ◀── 204 No Content                                                              │
│            Access-Control-Allow-Origin: http://localhost:3000                            │
│            Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS                    │
│            Access-Control-Allow-Headers: Content-Type, Authorization                           │
│                                                                                                     │
│   corsMiddleware answers OPTIONS directly and returns early — the real                              │
│   handler (createTaskHandler) never runs for this request at all.                                      │
└──────────────────────────────────────────────────────────┘
```

### Streaming (SSE)

```bash
curl -N http://localhost:8080/tasks/stream
```
(`-N` disables curl's output buffering, so you see each event as it
arrives, not all at once at the end.)
```
┌──────────────────────────────────────────────────────────┐
│   t=0s   data: {"tick": 0, "taskCount": 2}                          │
│                                                                          │
│   t=1s   data: {"tick": 1, "taskCount": 2}                                │
│                                                                              │
│   t=2s   data: {"tick": 2, "taskCount": 2}                                    │
│           ...                                                                    │
│   t=9s   data: {"tick": 9, "taskCount": 2}     ◀── connection closes after         │
│                                                     10 ticks                          │
│                                                                                           │
│   Open a SECOND terminal and create a task mid-stream (with the login/          │
│   POST steps above) — the NEXT tick's taskCount reflects it immediately,           │
│   since streamHandler reads live from the same store on every tick.                   │
└──────────────────────────────────────────────────────────┘
```

Press Ctrl+C partway through, and check the server's terminal — you won't
see any error or leftover goroutine warning, because `streamHandler`'s
`select` on `r.Context().Done()` (Module 13's propagation) notices the
disconnect and returns immediately instead of continuing to generate ticks
nobody will receive.

---

## 4. Sequence Diagram: A Complete Login → Create → Read Flow

```
   BROWSER/CURL                    REST SERVER                    TaskStore / SessionStore
        │                               │                                    │
        │  POST /login                     │                                    │
        │  {username, password}              │                                    │
        │──────────────────────────────────▶│                                    │
        │                               │  sessions.Create(username)              │
        │                               │───────────────────────────────────────▶│
        │                               │◀───────────────────────────────────────│
        │                               │  token                                    │
        │  Set-Cookie: session=<token>       │                                          │
        │◀──────────────────────────────────│                                          │
        │  (browser stores cookie)         │                                            │
        │                               │                                              │
        │  POST /tasks                     │                                              │
        │  Cookie: session=<token>            │                                              │
        │  {title: "Buy milk"}                  │                                              │
        │──────────────────────────────────▶│                                              │
        │                               │  sessions.Valid(token) ──▶ true                     │
        │                               │───────────────────────────────────────▶│              │
        │                               │◀───────────────────────────────────────│              │
        │                               │  store.Create("Buy milk")                                │
        │                               │───────────────────────────────────────▶│                    │
        │                               │◀───────────────────────────────────────│                    │
        │  201 Created                     │  new Task{ID:3, ...}                                        │
        │  {"id":3, "title":"Buy milk"}       │                                                              │
        │◀──────────────────────────────────│                                                              │
        │                               │                                                                    │
        │  GET /tasks                      │                                                                    │
        │  (no cookie needed — GET is public) │                                                                    │
        │──────────────────────────────────▶│                                                                    │
        │                               │  store.List()                                                          │
        │                               │───────────────────────────────────────▶│                              │
        │                               │◀───────────────────────────────────────│                              │
        │  200 OK  [Task{1}, Task{2}, Task{3}]  │                                                                    │
        │◀──────────────────────────────────│                                                                    │
```

---

## 5. Case Study: Why `authMiddleware` Checks the Method, Not the Path

```go
if r.Method == http.MethodGet || r.Method == http.MethodOptions {
	next.ServeHTTP(w, r)
	return
}
```

An alternative design would list *paths* requiring auth
(`"/tasks" if POST`, etc.) — but that gets more fragile as routes grow,
and it's easy to add a new mutating endpoint later and forget to add it to
an auth-required path list. Checking the **method** instead encodes a
simple, durable rule that holds regardless of how many routes exist: *"any
request that changes something requires a session; anything read-only
doesn't."* This is also why `OPTIONS` is explicitly let through here too —
without it, a CORS preflight (which is always `OPTIONS`, and browsers never
attach cookies to it in the way a real request would) could get blocked by
auth before `corsMiddleware` even gets a chance to answer it, breaking CORS
entirely for any protected route.

## Case Study: What Happens If You Reorder the Middleware Chain

Try swapping `corsMiddleware` and `authMiddleware`'s order in `main.go`'s
`chain(...)` call — putting auth *before* CORS:

```
┌──────────────────────────────────────────────────────────┐
│   Current (correct) order:        recover→log→reqID→CORS→auth→router    │
│                                                                              │
│   Swapped (broken) order:          recover→log→reqID→auth→CORS→router          │
│                                                                                    │
│   A browser's PREFLIGHT (OPTIONS) request would now hit authMiddleware              │
│   FIRST. It has no session cookie (preflights never carry one), so it                  │
│   gets rejected with 401 — and CORS headers are NEVER SET, because                        │
│   corsMiddleware never even runs. The browser then blocks the REAL                           │
│   request too, reporting a CORS error in the console — even though the                          │
│   actual root cause is an ordering bug in the middleware chain, not a                              │
│   CORS configuration problem at all.                                                                   │
└──────────────────────────────────────────────────────────┘
```

This is a genuinely common real-world debugging trap: a CORS error in the
browser console doesn't always mean the CORS *headers* are wrong — sometimes
it means something upstream of the CORS middleware is rejecting the
preflight before it ever gets a chance to answer.

---

## Try It Yourself

- Add a `GET /tasks/{id}/history` route and confirm `ServeMux` correctly
  prefers it over the more general `GET /tasks/{id}` for matching requests
- Change `corsMiddleware("http://localhost:3000")` to `corsMiddleware("*")`
  and think through what that trade-off means — wildcard origins can't be
  combined with cookies/credentials in real browsers, so what would actually
  break for this specific API if you made that change?
- Add a `DELETE /login` (logout) endpoint that removes the token from
  `SessionStore` and clears the cookie client-side by re-sending
  `Set-Cookie` with `MaxAge: -1`
- Open `http://localhost:8080/tasks/stream` directly in a browser tab (not
  curl) — browsers render SSE streams natively; watch the Network tab in
  dev tools to see each event arrive individually over one connection
