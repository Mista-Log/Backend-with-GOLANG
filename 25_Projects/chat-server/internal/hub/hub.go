// Package hub solves this project's hard part: safely broadcasting a
// message to many concurrently-connected WebSocket clients, while clients
// connect and disconnect at arbitrary moments. Rather than protecting a
// shared map of clients with a Mutex (which would work, but means every
// goroutine touching the client list needs to remember to lock), this
// uses Module 12's other core pattern instead: ONE goroutine (Run) owns
// the client registry exclusively, and everyone else talks to it only
// through channels — no shared memory to protect at all.
package hub

type Client struct {
	Send chan []byte // the Hub writes here; THIS client's own goroutine (see server package) reads it
	Name string
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func New() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

func (h *Hub) Register(c *Client)   { h.register <- c }
func (h *Hub) Unregister(c *Client) { h.unregister <- c }
func (h *Hub) Broadcast(msg []byte) { h.broadcast <- msg }

// Run is the ONE goroutine that ever touches h.clients directly — every
// other goroutine in this program communicates with it exclusively
// through the register/unregister/broadcast channels above. This is
// Module 12's "share memory by communicating" proverb, applied to the
// exact problem a naive Mutex-protected map would also solve, but
// without needing to reason about lock ordering or forgetting to unlock
// somewhere.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send) // signals that client's own writer goroutine to stop
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.Send <- msg: // non-blocking-enough: buffered channel, see server package
				default:
					// This client's send buffer is full — it's too slow or
					// stuck. Drop it rather than let one slow client block
					// the broadcast for everyone else (exactly the
					// semaphore/backpressure trade-off from Module 12's
					// Image Processor project, applied here as "disconnect
					// the straggler" instead of "slow everyone down").
					delete(h.clients, c)
					close(c.Send)
				}
			}
		}
	}
}
