# 15. HTTP

Everything before this module has used `net/http` in passing — `httptest`
servers in Modules 11-14, a JSON API in Module 06. This module is the real,
dedicated tour: how a request actually flows through a Go HTTP server, and
every piece you need to build a genuine, browser-facing API.

---

## 1. `net/http`

Go's HTTP server needs nothing beyond the standard library — no framework
required to get a real, production-capable server running.

```go
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, HTTP!")
	})
	http.ListenAndServe(":8080", nil)
}
```

```
┌──────────────────────────────────────────────────────────┐
│                     THE BIG PICTURE                              │
│                                                                       │
│   Browser / curl / fetch()                                              │
│         │  1. opens a TCP connection, sends an HTTP request               │
│         ▼                                                                    │
│   ┌─────────────────────────────────────────────┐                              │
│   │              Your Go program                    │                              │
│   │                                                   │                              │
│   │   http.ListenAndServe(":8080", handler)              │                              │
│   │         │                                              │                              │
│   │         ▼                                                │                              │
│   │   for each incoming request:                               │                              │
│   │     - a NEW GOROUTINE is spawned automatically                │  ◀── Module 12! Go's         │
│   │     - handler.ServeHTTP(w, r) runs on it                        │      HTTP server is           │
│   │         │                                                          │      concurrent by            │
│   │         ▼                                                            │      default, for FREE          │
│   │   your handler function writes to `w`                                   │                                │
│   └─────────────────────────────────────────────┘                                                    │
│         │  2. response bytes sent back over the SAME connection                                          │
│         ▼                                                                                                    │
│   Browser renders the page / your JS gets the response                                                          │
└──────────────────────────────────────────────────────────┘
```

**Every request runs on its own goroutine** — this is exactly why Module 12
mattered so much before this module: any shared state your handlers touch
(a database connection pool, an in-memory cache, a counter) needs the same
`Mutex`/`atomic`/channel protection you'd use anywhere else concurrent
goroutines share memory.

The core interface underneath everything:
```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```
`http.HandleFunc` is a convenience wrapper — any plain
`func(w http.ResponseWriter, r *http.Request)` automatically satisfies
`Handler` via the adapter type `http.HandlerFunc`. This is Module 06's
implicit interface satisfaction, one layer below everything else in this
module.

---

## 2. Request

`*http.Request` carries everything the client sent: method, URL, headers,
cookies, and body.

```
┌──────────────────────────────────────────────────────────┐
│         An incoming HTTP request, and where each piece lands in Go       │
│                                                                                │
│   POST /tasks?verbose=true HTTP/1.1              ┐                              │
│   Host: api.example.com                            │                             │
│   Content-Type: application/json                    ├─▶  r.Method   = "POST"       │
│   Cookie: session=abc123                              │   r.URL.Path = "/tasks"       │
│   X-Request-ID: req-42                                  │   r.URL.Query() = {"verbose":│
│                                                            │                "true"}      │
│   {"title": "Buy milk", "done": false}  ◀─────────────────┘   r.Header.Get("Content-Type")│
│         ▲                                                        r.Cookie("session")         │
│         │                                                        r.Body (an io.Reader —          │
│    the REQUEST BODY — read it with                                 STREAMED, Module 10 style)      │
│    json.NewDecoder(r.Body).Decode(&v)                                                                 │
└──────────────────────────────────────────────────────────┘
```

```go
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)              // "GET", "POST", ...
	fmt.Println(r.URL.Path)             // "/tasks/42"
	fmt.Println(r.URL.Query().Get("q"))  // query string parameter
	fmt.Println(r.Header.Get("Authorization"))

	var task Task
	json.NewDecoder(r.Body).Decode(&task) // streamed decode, Module 11's habit
}
```

**`r.Body` is a stream, not a byte slice** — exactly Module 10's `io.Reader`
pattern. It can only be read **once**; if middleware reads it, the handler
downstream gets nothing unless the middleware explicitly re-attaches a fresh
reader (see the Middleware section's case study).

---

## 3. Response

`http.ResponseWriter` is how a handler sends data back — note the order
things must happen in, since HTTP itself requires headers before the body:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // 1. set headers FIRST
	w.WriteHeader(http.StatusCreated)                     // 2. THEN the status code
	json.NewEncoder(w).Encode(task)                         // 3. THEN the body
}
```

```
┌──────────────────────────────────────────────────────────┐
│                  ResponseWriter's ONE-WAY DOOR                     │
│                                                                        │
│   w.Header().Set(...)   ──▶  buffered, can call this MANY times          │
│                                 — as long as nothing has been WRITTEN        │
│                                 yet                                              │
│                                                                                     │
│   w.WriteHeader(status)  ──▶  LOCKS IN the status code and headers,                   │
│         or                      sends them over the wire — happens                        │
│   w.Write(...) / first          IMPLICITLY on your first Write() call if                     │
│   Fprintf/Encode call            you never called WriteHeader yourself                          │
│                                   (defaults to 200 OK)                                              │
│                                                                                                          │
│   Calling w.Header().Set(...) AFTER this point has NO EFFECT — the                                        │
│   headers already went out. This is the single most common Go HTTP                                          │
│   bug for beginners: setting a header "too late."                                                              │
└──────────────────────────────────────────────────────────┘
```

---

## 4. Middleware

Middleware wraps a handler with cross-cutting behavior — logging, auth,
recovery, CORS — without the handler itself knowing it's there. In Go, a
middleware is simply **a function that takes a `Handler` and returns a
new one**: Module 06's composition and Module 03's higher-order functions,
applied directly to HTTP.

```go
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r) // call the WRAPPED handler
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}

handler := loggingMiddleware(myRealHandler)
```

```
┌──────────────────────────────────────────────────────────┐
│              A request flowing through a middleware CHAIN            │
│                                                                          │
│   incoming request                                                        │
│         │                                                                    │
│         ▼                                                                       │
│   ┌─────────────────┐                                                            │
│   │  recoverMW        │  ◀── outermost: catches panics from EVERYTHING inside      │
│   │   ┌─────────────┐  │                                                              │
│   │   │ loggingMW      │  │  ◀── times the WHOLE remaining chain, including CORS,       │
│   │   │  ┌──────────┐  │  │       auth, and the real handler                              │
│   │   │  │ corsMW      │  │  │                                                                  │
│   │   │  │  ┌───────┐  │  │  │                                                                    │
│   │   │  │  │ authMW    │  │  │  │  ◀── checks credentials BEFORE the real handler runs at all       │
│   │   │  │  │ ┌─────┐  │  │  │  │                                                                        │
│   │   │  │  │ │ REAL   │ │  │  │  │  ◀── your actual business logic, innermost                              │
│   │   │  │  │ │handler│ │  │  │  │                                                                            │
│   │   │  │  │ └─────┘  │  │  │  │                                                                                │
│   │   │  │  └───────┘  │  │  │                                                                                     │
│   │   │  └──────────┘  │  │                                                                                          │
│   │   └─────────────┘  │                                                                                                │
│   └─────────────────┘                                                                                                     │
│                                                                                                                                │
│   Execution goes IN through each layer, then back OUT in reverse —                                                                │
│   the exact LIFO shape as defer (Module 02) and the call stack (Module 05).                                                          │
└──────────────────────────────────────────────────────────┘
```

Chaining several middlewares reads inside-out, which gets hard to follow —
a small helper flattens it:
```go
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- { // apply LAST middleware innermost
		h = mws[i](h)
	}
	return h
}

handler := chain(realHandler, recoverMW, loggingMW, corsMW, authMW)
// reads top-to-bottom in the ORDER requests actually pass through them
```

**Case study — why reading `r.Body` in middleware needs care:** `r.Body` is
a stream (Section 2) that can only be drained once. A middleware that reads
it (to log the request body, say) must replace `r.Body` with a fresh reader
over the bytes it already consumed, or the real handler downstream gets an
empty body:
```go
body, _ := io.ReadAll(r.Body)
r.Body = io.NopCloser(bytes.NewReader(body)) // "rewind" it for the next handler
```

---

## 5. Routing

Go 1.22+'s standard `http.ServeMux` supports method-specific patterns and
path parameters natively — no third-party router required for most APIs
anymore:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /tasks", listTasks)
mux.HandleFunc("GET /tasks/{id}", getTask)
mux.HandleFunc("PUT /tasks/{id}", updateTask)
mux.HandleFunc("DELETE /tasks/{id}", deleteTask)

func getTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // extracts {id} from the matched pattern
	...
}
```

```
┌──────────────────────────────────────────────────────────┐
│                  How ServeMux picks a handler                       │
│                                                                          │
│   incoming: GET /tasks/42                                                 │
│                                                                               │
│   registered patterns:                                                          │
│     GET /tasks               ──▶ no match (path differs)                            │
│     GET /tasks/{id}           ──▶ MATCH — id="42"          ◀── MORE SPECIFIC          │
│     GET /tasks/{id}/comments   ──▶ no match                    patterns win over        │
│     GET /                       ──▶ would also technically      more general ones          │
│                                      match, but ServeMux                                        │
│                                      prefers the MOST                                              │
│                                      SPECIFIC match                                                    │
└──────────────────────────────────────────────────────────┘
```

For deeply nested routing, wildcard subtrees, or middleware-per-route-group
needs beyond what `ServeMux` covers, third-party routers (`chi`, `gorilla/mux`)
remain common in existing codebases — but for new code, the standard
library's own router now covers the large majority of real-world API
routing needs.

---

## 6. Cookies

Cookies are small key/value pairs the **browser stores and automatically
re-sends** on every subsequent request to the same domain — Go's
`http.Cookie` type reads and writes the `Cookie`/`Set-Cookie` headers for you.

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "abc123",
		Path:     "/",
		HttpOnly: true,                    // JS can't read it — mitigates XSS token theft
		Secure:   true,                     // only sent over HTTPS
		SameSite: http.SameSiteLaxMode,       // mitigates CSRF
		MaxAge:   3600,                        // seconds until expiry
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	fmt.Println("session value:", cookie.Value)
}
```

```
┌──────────────────────────────────────────────────────────┐
│                  Cookie round-trip across requests                   │
│                                                                          │
│   Request 1: POST /login                                                  │
│        Go server ──▶ Set-Cookie: session=abc123; HttpOnly; Secure           │
│        Browser stores it                                                       │
│                                                                                    │
│   Request 2: GET /tasks   (browser sends it back AUTOMATICALLY)                       │
│        Browser ──▶  Cookie: session=abc123                                              │
│        Go server reads it via r.Cookie("session")                                          │
│                                                                                                 │
│   This is why cookies are the classic mechanism for SESSIONS — the                                │
│   browser does the "remember and resend" work for you, on every                                     │
│   request, with zero JavaScript needed.                                                                │
└──────────────────────────────────────────────────────────┘
```

`HttpOnly`, `Secure`, and `SameSite` are all **security-relevant defaults
worth setting deliberately** on any cookie carrying a session token or
credential — omitting them doesn't break functionality in a browser, but it
does weaken real protections against common web attacks (XSS reading the
cookie via JS, or a cross-site request silently including it).

---

## 7. Headers

Headers carry metadata about the request/response — content type,
authentication, caching, custom application data.

```go
// Reading a request header:
auth := r.Header.Get("Authorization")

// Setting a response header (MUST happen before WriteHeader/Write — Section 3):
w.Header().Set("Content-Type", "application/json")
w.Header().Set("X-Request-ID", requestID)
w.Header().Add("Set-Cookie", cookie.String()) // Add, not Set, for MULTI-VALUE headers
```

```
┌──────────────────────────────────────────────────────────┐
│    Header().Get(key)  →  the FIRST value for that header name        │
│    Header().Set(key, v) →  REPLACES any existing values with ONE        │
│    Header().Add(key, v)  →  APPENDS an additional value (headers CAN        │
│                                legitimately repeat — Set-Cookie is the           │
│                                classic example: multiple cookies need              │
│                                multiple separate Set-Cookie header lines)             │
└──────────────────────────────────────────────────────────┘
```

**Content negotiation** — deciding the response format based on what the
client asked for — is just reading the `Accept` header and branching:
```go
if strings.Contains(r.Header.Get("Accept"), "application/xml") {
	xml.NewEncoder(w).Encode(data)
} else {
	json.NewEncoder(w).Encode(data) // default
}
```

---

## 8. CORS

**Cross-Origin Resource Sharing** — the browser-enforced security rule that
JavaScript on `https://mysite.com` **cannot** read a response from
`https://api.example.com` unless that response explicitly says it's allowed
to, via CORS headers. This is a **browser-only** restriction (curl, another
Go program, or a mobile app calling your API is completely unaffected by
CORS) — but it's the single most common source of confusion for anyone
building an API a browser-based frontend calls from a different domain.

```
┌──────────────────────────────────────────────────────────┐
│         Why a "preflight" OPTIONS request happens at all                │
│                                                                              │
│   Browser JS:  fetch("https://api.example.com/tasks", {                       │
│                    method: "POST", headers: {"Content-Type": "application/json"}│
│                })                                                                  │
│                                                                                        │
│   1. Browser sees this is a "non-simple" cross-origin request                            │
│      (custom header + POST) and sends a PREFLIGHT first:                                    │
│                                                                                                  │
│         OPTIONS /tasks HTTP/1.1                                                                    │
│         Origin: https://mysite.com                                                                    │
│         Access-Control-Request-Method: POST                                                              │
│         Access-Control-Request-Headers: Content-Type                                                        │
│                                                                                                                  │
│   2. Your Go server must respond to OPTIONS with what's ALLOWED:                                                    │
│                                                                                                                          │
│         Access-Control-Allow-Origin: https://mysite.com                                                                     │
│         Access-Control-Allow-Methods: GET, POST, PUT, DELETE                                                                    │
│         Access-Control-Allow-Headers: Content-Type                                                                                  │
│                                                                                                                                          │
│   3. ONLY IF the preflight response allows it does the browser send                                                                        │
│      the REAL POST request at all — if your server never answers                                                                              │
│      OPTIONS correctly, the browser blocks the real request before                                                                                │
│      it's even sent, and your handler never runs, and never even                                                                                     │
│      SEES the real request.                                                                                                                              │
└──────────────────────────────────────────────────────────┘
```

```go
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://mysite.com")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // answer the PREFLIGHT, don't run the real handler
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

**The single most common CORS mistake:** forgetting the early `return` after
answering `OPTIONS` — without it, the preflight request falls through into
your real handler logic too, which is at best wasted work and at worst
produces a body/side-effect for a request that was only ever meant to ask
"is this allowed?"

---

## 9. Streaming

A handler can write a response **incrementally**, flushing partial data to
the client before the full response is ready — essential for large
responses, and for **Server-Sent Events (SSE)**, a simple one-way
server-to-browser streaming protocol built entirely on a long-lived HTTP
response.

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	flusher, ok := w.(http.Flusher) // NOT every ResponseWriter supports flushing
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	for i := 0; i < 5; i++ {
		fmt.Fprintf(w, "data: update %d\n\n", i) // SSE's required "data: ...\n\n" format
		flusher.Flush()                             // push what's written so far, NOW
		time.Sleep(1 * time.Second)

		select {
		case <-r.Context().Done(): // client disconnected — Module 13's propagation
			return                    // in action: r.Context() is cancelled automatically
		default:
		}
	}
}
```

```
┌──────────────────────────────────────────────────────────┐
│              Normal response          vs.        Streamed (SSE) response       │
│                                                                                     │
│   server: [buffer everything]           server: write chunk 1 → FLUSH → sent NOW      │
│           write ALL at once                       write chunk 2 → FLUSH → sent NOW       │
│           client waits, then                        write chunk 3 → FLUSH → sent NOW        │
│           gets the WHOLE thing                        ...                                       │
│                                                            client receives EACH chunk               │
│                                                              as it arrives, over ONE LONG-             │
│                                                              LIVED connection, no polling                 │
└──────────────────────────────────────────────────────────┘
```

The browser-side JavaScript for consuming this needs no special library —
`EventSource` is built into every browser specifically for this format:
```javascript
const source = new EventSource("/stream");
source.onmessage = (event) => console.log(event.data);
```

**Note `r.Context().Done()` inside the loop** — this is Module 13's
propagation lesson directly applicable here: if the browser closes the tab
or navigates away, `r.Context()` is cancelled automatically, and a
well-written streaming handler checks for that and stops doing wasted work,
exactly like the buggy-vs-correct gateway comparison in Module 13's API
Timeout project.

---

Onto the project — REST Server ties every section above together into one
real, runnable API: routing, middleware chaining, cookies for session auth,
CORS for browser access, custom headers, and a streaming endpoint, all
against one small task-management resource.
