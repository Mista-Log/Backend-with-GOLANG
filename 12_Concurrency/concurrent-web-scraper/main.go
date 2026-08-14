// Project 1: Concurrent Web Scraper
//
// Scrapes a list of URLs concurrently against a local httptest server
// (fully offline, deterministic — some pages are deliberately slow, one is
// deliberately broken). Demonstrates a worker pool (fan-out), a fan-in
// merge of every worker's results into one channel, sync.Map for
// concurrent-safe URL deduplication, and context-based timeouts enforced
// with select.
package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type ScrapeResult struct {
	URL        string
	StatusCode int
	Bytes      int
	Duration   time.Duration
	Err        error
}

// --- Local server (stands in for the real internet) -----------------------

func startServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow":
			time.Sleep(300 * time.Millisecond) // slow enough to hit a per-request timeout
			fmt.Fprint(w, "<html>slow page</html>")
		case "/broken":
			http.Error(w, "internal error", http.StatusInternalServerError)
		case "/timeout":
			time.Sleep(5 * time.Second) // essentially never responds in time
			fmt.Fprint(w, "too late")
		default:
			time.Sleep(time.Duration(20+rand.Intn(80)) * time.Millisecond) // normal jitter
			fmt.Fprintf(w, "<html><body>content for %s, %d bytes of padding: %s</body></html>",
				r.URL.Path, 50, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		}
	})
	return httptest.NewServer(mux)
}

// --- Worker pool (fan-out) -----------------------------------------------

// scrapeOne fetches a single URL, enforcing a PER-REQUEST timeout via
// context — separate from the overall scrape deadline main sets up, so one
// slow page can't eat the whole budget meant for every other page.
func scrapeOne(ctx context.Context, client *http.Client, url string) ScrapeResult {
	start := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return ScrapeResult{URL: url, Err: err, Duration: time.Since(start)}
	}

	resp, err := client.Do(req)
	if err != nil {
		return ScrapeResult{URL: url, Err: err, Duration: time.Since(start)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ScrapeResult{URL: url, Err: err, Duration: time.Since(start)}
	}

	return ScrapeResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Bytes:      len(body),
		Duration:   time.Since(start),
	}
}

// worker is one fan-out branch: pulls URLs off a shared channel until it's
// closed, skipping anything already visited (tracked in a sync.Map shared
// safely across every worker with no explicit locking needed).
func worker(ctx context.Context, id int, urls <-chan string, results chan<- ScrapeResult, visited *sync.Map, client *http.Client) {
	for url := range urls {
		if _, alreadySeen := visited.LoadOrStore(url, true); alreadySeen {
			continue // another worker already claimed this URL — skip it
		}
		results <- scrapeOne(ctx, client, url)
	}
}

func main() {
	server := startServer()
	defer server.Close()

	urls := []string{
		server.URL + "/page1",
		server.URL + "/page2",
		server.URL + "/page1", // deliberate duplicate — sync.Map should dedup this
		server.URL + "/page3",
		server.URL + "/slow",
		server.URL + "/broken",
		server.URL + "/timeout", // will hit the per-request timeout
		server.URL + "/page4",
	}

	const numWorkers = 3
	ctx, overallCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer overallCancel()

	urlCh := make(chan string)
	resultsCh := make(chan ScrapeResult)
	var visited sync.Map
	client := &http.Client{}

	// Fan-OUT: numWorkers goroutines all read from the same urlCh.
	var workerWG sync.WaitGroup
	for i := 1; i <= numWorkers; i++ {
		workerWG.Add(1)
		go func(id int) {
			defer workerWG.Done()
			worker(ctx, id, urlCh, resultsCh, &visited, client)
		}(i)
	}

	// Feed URLs in, then close urlCh so workers know to stop once drained.
	go func() {
		for _, u := range urls {
			urlCh <- u
		}
		close(urlCh)
	}()

	// Fan-IN happens implicitly here: resultsCh IS the merge point, since
	// every worker writes to the SAME results channel. Closing it once all
	// workers finish is what makes the `for range resultsCh` below terminate.
	go func() {
		workerWG.Wait()
		close(resultsCh)
	}()

	fmt.Printf("Scraping %d URLs (with 1 duplicate) using %d workers...\n\n", len(urls), numWorkers)

	successCount, errorCount := 0, 0
	for result := range resultsCh {
		if result.Err != nil {
			errorCount++
			fmt.Printf("  FAILED  %-45s %v\n", result.URL, result.Err)
			continue
		}
		if result.StatusCode >= 400 {
			errorCount++
			fmt.Printf("  FAILED  %-45s status=%d\n", result.URL, result.StatusCode)
			continue
		}
		successCount++
		fmt.Printf("  OK      %-45s status=%d bytes=%d took=%v\n",
			result.URL, result.StatusCode, result.Bytes, result.Duration.Round(time.Millisecond))
	}

	fmt.Printf("\nDone: %d succeeded, %d failed (duplicates were skipped, not counted as either)\n", successCount, errorCount)
}
