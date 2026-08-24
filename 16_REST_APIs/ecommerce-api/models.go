package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"inStock"`
}

type ProductStore struct {
	mu       sync.RWMutex
	products map[int]*Product
	nextID   int
}

func NewProductStore() *ProductStore {
	return &ProductStore{products: make(map[int]*Product), nextID: 1}
}

func (s *ProductStore) Create(p Product) *Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID
	s.products[p.ID] = &p
	s.nextID++
	return &p
}

func (s *ProductStore) Get(id int) (*Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *ProductStore) Update(id int, p Product) (*Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.products[id]
	if !ok {
		return nil, fmt.Errorf("no product with id %d", id)
	}
	p.ID = id
	*existing = p
	return existing, nil
}

func (s *ProductStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return fmt.Errorf("no product with id %d", id)
	}
	delete(s.products, id)
	return nil
}

// All returns every product, unfiltered, unsorted — filtering/sorting/
// pagination all happen in the handler layer (query.go), so this stays a
// simple, reusable building block.
func (s *ProductStore) All() []*Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Search does a case-insensitive substring match across Name and
// Description — the guide's Search section, applied directly.
func (s *ProductStore) Search(query string) []*Product {
	query = strings.ToLower(query)
	all := s.All()
	var results []*Product
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			results = append(results, p)
		}
	}
	return results
}
