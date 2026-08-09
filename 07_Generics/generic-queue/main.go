// Project 1: Generic Queue
//
// A single FIFO queue implementation, usable with ints, strings, or any
// custom struct — written once, with full compile-time type safety and zero
// type assertions needed by callers.
package main

import "fmt"

// Queue[T any] takes NO constraint beyond `any`, because a queue never
// needs to compare, add, or otherwise operate ON its elements — it only
// ever stores and returns them. That's the signal for when `any` (rather
// than a narrower constraint) is the right choice, per the guide.
type Queue[T any] struct {
	items []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

// Enqueue adds to the back.
func (q *Queue[T]) Enqueue(item T) {
	q.items = append(q.items, item)
}

// Dequeue removes and returns the front item. The "comma ok" return lets
// callers distinguish "got a real (possibly zero-valued) item" from
// "queue was empty" — same shape as map lookups and channel receives.
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	front := q.items[0]
	q.items = q.items[1:]
	return front, true
}

func (q *Queue[T]) Peek() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	return q.items[0], true
}

func (q *Queue[T]) Len() int {
	return len(q.items)
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}

// --- Demo -----------------------------------------------------

// Task is an ordinary struct — nothing special needs to be done to it for
// Queue[Task] to work below. That's the whole point of `[T any]`.
type Task struct {
	Name     string
	Priority int
}

func main() {
	fmt.Println("=== Queue[int] ===")
	intQueue := NewQueue[int]()
	intQueue.Enqueue(1)
	intQueue.Enqueue(2)
	intQueue.Enqueue(3)
	for !intQueue.IsEmpty() {
		val, _ := intQueue.Dequeue()
		fmt.Println("dequeued:", val) // prints 1, 2, 3 — FIFO order
	}

	fmt.Println("\n=== Queue[string] ===")
	stringQueue := NewQueue[string]()
	stringQueue.Enqueue("first")
	stringQueue.Enqueue("second")
	val, ok := stringQueue.Dequeue()
	fmt.Println("dequeued:", val, ok)

	fmt.Println("\n=== Queue[Task] — a custom struct, no extra code needed ===")
	taskQueue := NewQueue[Task]()
	taskQueue.Enqueue(Task{Name: "Deploy", Priority: 1})
	taskQueue.Enqueue(Task{Name: "Write docs", Priority: 3})
	taskQueue.Enqueue(Task{Name: "Fix bug", Priority: 2})

	fmt.Println("Processing tasks in FIFO order:")
	for !taskQueue.IsEmpty() {
		task, _ := taskQueue.Dequeue()
		fmt.Printf("  processing %q (priority %d)\n", task.Name, task.Priority)
	}

	fmt.Println("\n=== Dequeue on an empty queue ===")
	empty := NewQueue[int]()
	zeroVal, ok := empty.Dequeue()
	fmt.Printf("value: %v, ok: %v (zero value for int is 0, NOT an error by itself)\n", zeroVal, ok)
}
