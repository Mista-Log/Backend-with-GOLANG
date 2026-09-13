// Package server handles the actual WebSocket connections. Each connected
// client gets TWO goroutines (Module 12): one reading incoming messages
// (readPump) and one writing outgoing messages (writePump) — split
// specifically because gorilla/websocket's Conn is not safe for
// concurrent reads OR concurrent writes from multiple goroutines, but IS
// safe for one reader and one writer running at the same time.
package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"chatserver/internal/hub"
)

// CheckOrigin always returns true here for a local demo — a real
// deployment should validate Origin against a known frontend domain.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// readPump reads every message THIS client sends, and hands it to the
// Hub to broadcast to everyone else. It owns the connection's READ side
// exclusively.
func readPump(conn *websocket.Conn, h *hub.Hub, client *hub.Client) {
	defer func() {
		h.Unregister(client)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return // client disconnected, or a real error — either way, stop
		}
		formatted := []byte(fmt.Sprintf("%s: %s", client.Name, message))
		h.Broadcast(formatted)
	}
}

// writePump sends every message the Hub queued for THIS client, over the
// connection. It owns the connection's WRITE side exclusively — this
// split is exactly why the Hub never writes to conn directly, only to
// client.Send, which ONLY this goroutine ever reads from.
func writePump(conn *websocket.Conn, client *hub.Client) {
	defer conn.Close()
	for message := range client.Send { // exits automatically when Hub closes this channel
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func NewHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "anonymous"
		}

		conn, err := upgrader.Upgrade(w, r, nil) // the ONE HTTP request that becomes a WebSocket
		if err != nil {
			log.Println("upgrade failed:", err)
			return
		}

		client := &hub.Client{Send: make(chan []byte, 16), Name: name} // buffered — see hub.go's backpressure note
		h.Register(client)
		h.Broadcast([]byte(fmt.Sprintf("* %s joined the chat", name)))

		// readPump blocks the REQUEST goroutine http.Server already gave us
		// (Module 15: every request runs on its own goroutine) — writePump
		// needs its OWN, separate goroutine, since both run concurrently
		// for the lifetime of this one connection.
		go writePump(conn, client)
		readPump(conn, h, client)
	}
}
