// Project 2: API Timeout
//
// Three real HTTP hops — client -> gateway -> downstream service, all local
// via httptest — comparing a gateway handler that correctly PROPAGATES the
// incoming request's context (so cancellation flows all the way down) against
// one that doesn't (a real, common bug: using context.Background() for an
// outbound call instead of deriving from the inbound request's context).
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// --- Downstream service -----------------------------------------------

// startDownstream simulates a slow external API. Critically, it respects
// ITS OWN r.Context() — the third hop in the propagation chain. If the
// gateway's outbound request context gets cancelled, THIS handler notices
// and stops too, instead of blindly sleeping the full delay regardless.
func startDownstream() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(300 * time.Millisecond): // simulated slow upstream work
			fmt.Fprint(w, `{"temp": 72}`)
		case <-r.Context().Done():
			fmt.Println("    [downstream] request cancelled mid-flight — stopped, no wasted work")
			return
		}
	})
	return httptest.NewServer(mux)
}

func callDownstream(ctx context.Context, downstreamURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downstreamURL+"/weather", nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// --- Gateway: one CORRECT handler, one BUGGY handler ------------------

func startGateway(downstreamURL string, wg *sync.WaitGroup) *httptest.Server {
	mux := http.NewServeMux()

	// CORRECT: derives its own timeout FROM r.Context() — the incoming
	// request's context. If the client disconnects (or the client's own
	// timeout fires) net/http cancels r.Context() automatically, and that
	// cancellation flows straight through to the downstream call too.
	mux.HandleFunc("/forecast", func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), 150*time.Millisecond)
		defer cancel()

		status, err := callDownstream(ctx, downstreamURL)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("  [gateway/good] gave up after %v: %v\n", elapsed.Round(time.Millisecond), err)
			http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
			return
		}
		fmt.Printf("  [gateway/good] downstream responded (%d) after %v\n", status, elapsed.Round(time.Millisecond))
		fmt.Fprint(w, "ok")
	})

	// BUGGY: uses context.Background() for the outbound call — completely
	// severing it from r.Context(). No timeout of its own either. When the
	// client gives up, THIS handler has no way to find out — it just keeps
	// running, holding a goroutine and a connection, for the FULL downstream
	// delay, producing a response nobody is listening for anymore.
	mux.HandleFunc("/forecast-buggy", func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()
		start := time.Now()

		ctx := context.Background() // 🐛 THE BUG — ignores r.Context() entirely

		status, err := callDownstream(ctx, downstreamURL)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("  [gateway/buggy] error after %v: %v\n", elapsed.Round(time.Millisecond), err)
			return
		}
		fmt.Printf("  [gateway/buggy] downstream responded (%d) after %v — but the client gave up LONG before this (see below)\n",
			status, elapsed.Round(time.Millisecond))
		fmt.Fprint(w, "ok") // almost certainly writing to a client that's already gone
	})

	return httptest.NewServer(mux)
}

// --- Client -----------------------------------------------------

func callGateway(path, gatewayURL string, clientTimeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gatewayURL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("[client] %-16s gave up after %v (client budget was %v): %v\n", path, elapsed.Round(time.Millisecond), clientTimeout, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[client] %-16s got a response after %v: status %d\n", path, elapsed.Round(time.Millisecond), resp.StatusCode)
}

func main() {
	downstream := startDownstream()
	defer downstream.Close()

	var handlerWG sync.WaitGroup
	gateway := startGateway(downstream.URL, &handlerWG)
	defer gateway.Close()

	// Deliberately short — shorter than downstream's 300ms delay, so the
	// CLIENT always gives up first on both paths. What differs is whether
	// the SERVER-SIDE handler notices and stops too.
	const clientBudget = 100 * time.Millisecond

	fmt.Println("=== Calling the CORRECT gateway endpoint (propagates r.Context()) ===")
	callGateway("/forecast", gateway.URL, clientBudget)

	fmt.Println("\n=== Calling the BUGGY gateway endpoint (uses context.Background()) ===")
	callGateway("/forecast-buggy", gateway.URL, clientBudget)

	fmt.Println("\nBoth client calls have already returned. Waiting to see what the")
	fmt.Println("server-side handlers do next (this is the part that matters)...")
	handlerWG.Wait() // block until BOTH handlers have actually finished server-side

	fmt.Println("\n=== Summary ===")
	fmt.Println("The CORRECT handler's downstream call was cancelled almost immediately")
	fmt.Println("once the client gave up — no wasted server-side work.")
	fmt.Println("The BUGGY handler kept running for the full ~300ms regardless, holding a")
	fmt.Println("goroutine and an outbound connection open for a response nobody could use.")
}
