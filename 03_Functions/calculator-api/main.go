// Project 2: Calculator API
//
// A small HTTP JSON API. Every endpoint is powered by the function concepts
// from this module: a dispatch table of functions-as-values, a higher-order
// middleware function that wraps handlers, a closure that keeps a running
// request counter, a variadic sum endpoint, and a recursive power endpoint.
//
// Run with: go run .
// Then in another terminal:
//   curl "http://localhost:8080/calculate?op=add&a=4&b=5"
//   curl "http://localhost:8080/sum?n=1&n=2&n=3&n=4"
//   curl "http://localhost:8080/power?base=2&exp=10"
//   curl "http://localhost:8080/"
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// --- Dispatch table ---------------------------------------------------
//
// operations maps an operation name to a FUNCTION VALUE. This is the same
// idea as mathlib's Map/Filter/Reduce (functions as data), applied to
// routing: adding a new operation is a one-line map entry, not a new
// `case` in a growing switch statement.
var operations = map[string]func(a, b float64) (float64, error){
	"add": func(a, b float64) (float64, error) { return a + b, nil },
	"sub": func(a, b float64) (float64, error) { return a - b, nil },
	"mul": func(a, b float64) (float64, error) { return a * b, nil },
	"div": func(a, b float64) (float64, error) {
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	},
}

// --- Response helpers ---------------------------------------------------

type response struct {
	Result float64 `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// --- Middleware (a higher-order function) -------------------------------
//
// withLogging takes a handler and RETURNS a new handler that wraps it —
// the textbook definition of a higher-order function, and the same pattern
// net/http itself uses everywhere (a "middleware" in Go is just a function
// of type func(http.Handler) http.Handler).
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r) // call the wrapped handler
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	}
}

// withRequestCount is a CLOSURE-producing middleware: `count` is captured
// by the returned handler and persists across every request that flows
// through it, protected by a mutex since the HTTP server handles requests
// on multiple goroutines concurrently.
func withRequestCount(next http.HandlerFunc) http.HandlerFunc {
	var (
		mu    sync.Mutex
		count int
	)
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		current := count
		mu.Unlock()

		w.Header().Set("X-Request-Count", strconv.Itoa(current))
		next(w, r)
	}
}

// --- Handlers ---------------------------------------------------

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	op := r.URL.Query().Get("op")
	fn, known := operations[op] // dispatch table lookup instead of a switch
	if !known {
		writeJSON(w, http.StatusBadRequest, response{Error: fmt.Sprintf("unknown operation %q", op)})
		return
	}

	a, errA := strconv.ParseFloat(r.URL.Query().Get("a"), 64)
	b, errB := strconv.ParseFloat(r.URL.Query().Get("b"), 64)
	if errA != nil || errB != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "a and b must both be numbers"})
		return
	}

	result, err := fn(a, b)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{Result: result})
}

// sumOf is VARIADIC — reused directly from the same idea as mathlib.Sum.
func sumOf(nums ...float64) float64 {
	total := 0.0
	for _, n := range nums {
		total += n
	}
	return total
}

func sumHandler(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query()["n"] // repeated ?n=1&n=2&n=3 query params
	if len(raw) == 0 {
		writeJSON(w, http.StatusBadRequest, response{Error: "provide at least one ?n= value"})
		return
	}

	nums := make([]float64, 0, len(raw))
	for _, s := range raw {
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{Error: fmt.Sprintf("invalid number %q", s)})
			return
		}
		nums = append(nums, n)
	}

	writeJSON(w, http.StatusOK, response{Result: sumOf(nums...)})
}

// power is RECURSIVE: base^exp = base * base^(exp-1), base case exp == 0.
// Only handles non-negative integer exponents, which keeps the recursion
// simple and finite.
func power(base float64, exp int) (float64, error) {
	if exp < 0 {
		return 0, fmt.Errorf("negative exponents are not supported")
	}
	if exp == 0 {
		return 1, nil
	}
	rest, err := power(base, exp-1)
	if err != nil {
		return 0, err
	}
	return base * rest, nil
}

func powerHandler(w http.ResponseWriter, r *http.Request) {
	base, errBase := strconv.ParseFloat(r.URL.Query().Get("base"), 64)
	exp, errExp := strconv.Atoi(r.URL.Query().Get("exp"))
	if errBase != nil || errExp != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "base must be a number, exp must be an integer"})
		return
	}

	result, err := power(base, exp)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{Result: result})
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"calculate": "/calculate?op=add|sub|mul|div&a=<num>&b=<num>",
		"sum":       "/sum?n=<num>&n=<num>...",
		"power":     "/power?base=<num>&exp=<non-negative int>",
	})
}

func main() {
	mux := http.NewServeMux()

	// Every route is wrapped by BOTH middleware functions — composing
	// higher-order functions is exactly how real Go web frameworks chain
	// logging, auth, rate-limiting, etc. around a plain handler.
	mux.HandleFunc("/", withLogging(withRequestCount(homeHandler)))
	mux.HandleFunc("/calculate", withLogging(withRequestCount(calculateHandler)))
	mux.HandleFunc("/sum", withLogging(withRequestCount(sumHandler)))
	mux.HandleFunc("/power", withLogging(withRequestCount(powerHandler)))

	addr := ":8080"
	fmt.Println("Calculator API listening on http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
