package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// listProductsHandler is the ONE endpoint doing the most work in this
// project: filter -> sort -> paginate -> (maybe) cache. Chi handlers are
// PLAIN net/http functions (Module 15), unlike Gin's framework-owned
// Context — this is the contrast the guide's Frameworks section sets up.
func listProductsHandler(store *ProductStore, cache *QueryCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cacheKey := r.URL.RawQuery
		if cacheKey == "" {
			cacheKey = "(no query)"
		}

		if cached, ok := cache.Get(cacheKey); ok {
			w.Header().Set("X-Cache", "HIT")
			writeJSON(w, http.StatusOK, cached)
			return
		}

		q := r.URL.Query()
		products := filterProducts(store.All(), q)
		sortProducts(products, q.Get("sort"), q.Get("order"))

		page, _ := strconv.Atoi(q.Get("page"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		result := paginate(products, page, limit)

		cache.Set(cacheKey, result)
		w.Header().Set("X-Cache", "MISS")
		writeJSON(w, http.StatusOK, result)
	}
}

// searchProductsHandler is intentionally NOT cached — search queries have
// far higher cardinality (nearly every query string is unique) than the
// filter/sort/paginate combinations on the main list endpoint, so caching
// would mostly just fill memory with one-time entries and rarely hit.
func searchProductsHandler(store *ProductStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			writeError(w, http.StatusBadRequest, "missing ?q= search query")
			return
		}
		results := store.Search(query)
		writeJSON(w, http.StatusOK, map[string]any{"query": query, "results": results, "count": len(results)})
	}
}

func getProductHandler(store *ProductStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product id")
			return
		}
		product, ok := store.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeJSON(w, http.StatusOK, product)
	}
}

func createProductHandler(store *ProductStore, cache *QueryCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if p.Name == "" || p.Price <= 0 {
			writeError(w, http.StatusBadRequest, "name is required and price must be positive")
			return
		}
		created := store.Create(p)
		cache.Invalidate() // a new product could appear in ANY cached listing — clear it all
		writeJSON(w, http.StatusCreated, created)
	}
}

func updateProductHandler(store *ProductStore, cache *QueryCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product id")
			return
		}
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		updated, err := store.Update(id, p)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		cache.Invalidate()
		writeJSON(w, http.StatusOK, updated)
	}
}

func deleteProductHandler(store *ProductStore, cache *QueryCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product id")
			return
		}
		if err := store.Delete(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		cache.Invalidate()
		w.WriteHeader(http.StatusNoContent)
	}
}
