# Intermediate — Chat Server (WebSockets)

```bash
cd chat-server
go mod tidy
go run .
```
Open `http://localhost:8080/?name=ada` and `http://localhost:8080/?name=bob`
in two browser tabs and chat between them in real time.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│   client A ──WebSocket──▶ readPump(A) ──▶ h.Broadcast(msg)              │
│                                                    │                        │
│                                                    ▼                          │
│                                              Hub.Run() (ONE goroutine,           │
│                                                owns the client map               │
│                                                exclusively)                        │
│                                                    │                                  │
│                              ┌─────────────────────┼─────────────────────┐             │
│                              ▼                     ▼                     ▼             │
│                        client A.Send        client B.Send        client C.Send           │
│                              │                     │                     │                 │
│                              ▼                     ▼                     ▼                   │
│                        writePump(A)          writePump(B)          writePump(C)                │
│                              │                     │                     │                       │
│                              ▼                     ▼                     ▼                         │
│                        WebSocket A           WebSocket B           WebSocket C                        │
└──────────────────────────────────────────────────────────┘
```

Every client gets exactly two goroutines (`readPump`, `writePump`) plus a
buffered channel (`Send`) the Hub writes into and only `writePump` reads
from — this is what lets `gorilla/websocket`'s one-reader/one-writer rule
hold safely per connection, while the Hub itself never touches a
connection directly at all.

## Case Study: Why a Central Hub Goroutine Instead of a Mutex-Protected Map

Either approach is correct. A `sync.Mutex`-protected `map[*Client]bool`
would work fine here too. The Hub approach is worth knowing as the
*other* idiomatic option specifically because it scales better once the
registry needs to do more than just "add/remove/iterate" — a Hub can
easily grow features (per-room broadcasting, rate limiting per client,
presence tracking) as new `case` branches in its `select`, all still
single-threaded from the Hub's own point of view, with zero new locking
to reason about. A Mutex-protected map tends to get harder to extend
safely as the logic inside each locked section grows.

## Try It Yourself
- Add "rooms" — a `map[string]*Hub`, one Hub per room, so messages only
  broadcast to clients in the same room
- Add a `/history` endpoint returning the last 50 messages (a ring buffer,
  written to inside `Hub.Run` alongside broadcasting)
- Replace the buffered-channel-drop-on-full backpressure policy with a
  bounded retry-with-timeout instead, and discuss the trade-off: is it
  better to drop a slow client's messages, or to make everyone else wait
  for them?
