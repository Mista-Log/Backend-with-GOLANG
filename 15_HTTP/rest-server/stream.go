package main

import (
	"fmt"
	"net/http"
	"time"
)

// streamHandler pushes the current task count to the client every second
// via Server-Sent Events, for up to 10 updates — matching Section 9's
// guide example, now wired to real, live data from the store.
func streamHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "data: {\"tick\": %d, \"taskCount\": %d}\n\n", i, store.Count())
			flusher.Flush()

			select {
			case <-time.After(1 * time.Second):
				// continue to the next tick
			case <-r.Context().Done():
				// The browser tab was closed, or the client disconnected —
				// Module 13's propagation lesson: stop immediately instead
				// of continuing to generate updates nobody will receive.
				return
			}
		}
	}
}
