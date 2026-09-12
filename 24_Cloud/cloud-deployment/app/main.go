// main.go is the ONE Go service this entire module deploys — deliberately
// small, so every surrounding piece (Docker, Compose, Kubernetes, Nginx,
// Terraform, CI/CD, Prometheus, Grafana) stays the focus. Its /healthz,
// /readyz, and /metrics endpoints match Module 19's contract exactly,
// since that contract is what every deployment tool in this module
// expects to find.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests."},
	[]string{"path", "status"},
)

func init() {
	prometheus.MustRegister(requestsTotal)
}

func withMetrics(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
		requestsTotal.WithLabelValues(path, "200").Inc()
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", withMetrics("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from cloudapp, running on port %s\n", port)
	}))

	// LIVENESS — matches Module 19's contract: is the process alive at all?
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// READINESS — matches Module 19's contract: ready for real traffic?
	// Always healthy here, since there's no real dependency to check —
	// the SHAPE is what Kubernetes' readinessProbe actually cares about.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("cloudapp listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
