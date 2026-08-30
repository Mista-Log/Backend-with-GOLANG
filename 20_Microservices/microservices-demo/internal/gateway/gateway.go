// Package gateway is the single external entry point — the guide's API
// Gateway diagram, implemented for real. External clients only ever talk
// to the gateway; it looks up the right backend service via the registry
// and reverse-proxies the request there.
package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"microservicesdemo/internal/registry"
)

func NewHandler(reg *registry.Registry) http.Handler {
	mux := http.NewServeMux()

	// Services call this on startup to announce where they're listening —
	// a simple form of SELF-REGISTRATION (as opposed to the gateway/registry
	// actively polling for services, another valid discovery strategy).
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Service string `json:"service"`
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		reg.Register(req.Service, req.Address)
		log.Printf("[gateway] registered %q at %s", req.Service, req.Address)
		w.WriteHeader(http.StatusNoContent)
	})

	// Catch-all: /api/{service}/... is proxied to whatever "{service}" the
	// registry currently has an address for. External clients never see
	// order-service's real address at all — only the gateway's.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/") // e.g. "orders/42"
		parts := strings.SplitN(path, "/", 2)
		serviceName := parts[0] + "-service" // "orders" -> "orders-service"

		addr, err := reg.Discover(serviceName)
		if err != nil {
			http.Error(w, "service unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}

		target, _ := url.Parse("http://" + addr)
		proxy := httputil.NewSingleHostReverseProxy(target)

		// The backend service sees a normal "/orders/42" path, exactly as
		// if it were called directly — it doesn't need to know it's behind
		// a gateway at all.
		r.URL.Path = "/" + path

		log.Printf("[gateway] proxying %s /api/%s -> %s%s", r.Method, path, addr, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	return mux
}
