package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

// TodoStore is a Mutex-protected map (Module 12) — Gin runs each request on
// its own goroutine, same as stdlib net/http (Module 15), so shared state
// needs the same protection here as anywhere else.
type TodoStore struct {
	mu     sync.Mutex
	todos  map[int]*Todo
	nextID int
}

func NewTodoStore() *TodoStore {
	return &TodoStore{todos: make(map[int]*Todo), nextID: 1}
}

func (s *TodoStore) Create(title string) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Todo{ID: s.nextID, Title: title, CreatedAt: time.Now()}
	s.todos[t.ID] = t
	s.nextID++
	return t
}

func (s *TodoStore) Get(id int) (*Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	return t, ok
}

// List returns a PAGINATED slice, plus the total count across ALL todos —
// callers need the total to compute totalPages for the response envelope.
func (s *TodoStore) List(page, limit int) (results []*Todo, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	total = len(all)
	offset := (page - 1) * limit
	if offset >= total || offset < 0 {
		return []*Todo{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total
}

// Replace implements PUT semantics — every field is overwritten, including
// resetting Done to whatever the request says (even false), matching the
// guide's PUT-vs-PATCH distinction exactly.
func (s *TodoStore) Replace(id int, title string, done bool) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("no todo with id %d", id)
	}
	t.Title = title
	t.Done = done
	return t, nil
}

// PatchDone implements PATCH semantics — only touches the ONE field it was
// given, leaving Title (and everything else) exactly as it was.
func (s *TodoStore) PatchDone(id int, done bool) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("no todo with id %d", id)
	}
	t.Done = done
	return t, nil
}

func (s *TodoStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.todos[id]; !ok {
		return fmt.Errorf("no todo with id %d", id)
	}
	delete(s.todos, id)
	return nil
}
