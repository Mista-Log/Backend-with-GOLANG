package main

import (
	"fmt"
	"sort"
	"sync"
)

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// TaskStore is a plain map protected by an RWMutex (Module 12) — reads
// (listing, getting) can happen concurrently; writes (create/update/delete)
// need exclusive access.
type TaskStore struct {
	mu     sync.RWMutex
	tasks  map[int]*Task
	nextID int
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[int]*Task), nextID: 1}
}

func (s *TaskStore) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *TaskStore) Get(id int) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *TaskStore) Create(title string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Task{ID: s.nextID, Title: title, Done: false}
	s.tasks[t.ID] = t
	s.nextID++
	return t
}

func (s *TaskStore) Update(id int, title string, done bool) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("no task with id %d", id)
	}
	t.Title = title
	t.Done = done
	return t, nil
}

func (s *TaskStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("no task with id %d", id)
	}
	delete(s.tasks, id)
	return nil
}

func (s *TaskStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}
