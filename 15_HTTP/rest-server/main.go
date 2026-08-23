// REST Server — ties together every section of Module 15's guide: routing
// (Go 1.22+ method+path patterns), request/response handling, a chained
// middleware stack, cookie-based session auth, CORS for browser access, and
// an SSE streaming endpoint — all against one small task-management API.
//
// Run with: go run .
// Then see README.md for a full curl walkthrough.
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	store := NewTaskStore()
	store.Create("Write the HTTP module guide")
	store.Create("Build the REST server project")
	sessions := NewSessionStore()

	mux := http.NewServeMux()

	// --- Routing (Section 5): method + path patterns, path parameters ---
	mux.HandleFunc("POST /login", loginHandler(sessions))
	mux.HandleFunc("GET /tasks", listTasksHandler(store))
	mux.HandleFunc("GET /tasks/stream", streamHandler(store))
	mux.HandleFunc("GET /tasks/{id}", getTaskHandler(store))
	mux.HandleFunc("POST /tasks", createTaskHandler(store))
	mux.HandleFunc("PUT /tasks/{id}", updateTaskHandler(store))
	mux.HandleFunc("DELETE /tasks/{id}", deleteTaskHandler(store))

	// --- Middleware chain (Section 4) ---
	// Order matters — see the README's diagram for exactly how a request
	// flows through this list, outermost first.
	handler := chain(mux,
		recoverMiddleware,             // outermost: catches panics from EVERYTHING below
		loggingMiddleware,               // times the whole remaining chain
		requestIDMiddleware,               // tags the request/response
		corsMiddleware("http://localhost:3000"), // answers preflight, sets CORS headers
		authMiddleware(sessions),             // innermost: protects mutating routes only
	)

	const addr = ":8080"
	fmt.Println("REST server listening on http://localhost" + addr)
	fmt.Println("See README.md for a full curl walkthrough.")
	log.Fatal(http.ListenAndServe(addr, handler))
}
