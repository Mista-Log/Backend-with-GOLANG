// main.go is the actual REST API this module optimizes — it serves the
// OPTIMIZED handlers (unoptimized/ is kept only for the benchmark
// comparison in bench_test.go), with net/http/pprof wired in on a
// separate port for live profiling, per the guide's pprof section.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // side-effect import: registers /debug/pprof/* on http.DefaultServeMux

	"optimizerestapi/optimized"
	"optimizerestapi/shared"
)

func main() {
	products := shared.SampleProducts(5000)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")
		var result []shared.Product
		if category != "" {
			result = optimized.FilterByCategory(products, category)
		} else {
			result = products
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("GET /products/report.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(optimized.BuildCSVReport(products)))
	})

	mux.HandleFunc("GET /products/search", func(w http.ResponseWriter, r *http.Request) {
		pattern := r.URL.Query().Get("pattern")
		matches, err := optimized.SearchByNamePattern(products, pattern)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matches)
	})

	// pprof runs on its OWN port, deliberately never exposed alongside the
	// real API — the guide's pprof section calls this out explicitly.
	go func() {
		log.Println("pprof listening on http://localhost:6060/debug/pprof/")
		http.ListenAndServe("localhost:6060", nil)
	}()

	fmt.Println("REST API listening on http://localhost:8080")
	fmt.Println("Try: curl 'http://localhost:8080/products/report.csv'")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
