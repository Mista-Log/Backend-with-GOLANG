// Project 2: Generic Stack
//
// A LIFO stack, same shape as the Generic Queue but reversed. Also
// introduces a CUSTOM CONSTRAINT (Ordered) for a standalone function that
// needs to compare elements, which Stack[T any] alone can't support.
package main

import "fmt"

type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Items returns a COPY of the stack's contents, bottom to top — using
// copy() here rather than returning s.items directly, same "don't leak the
// backing array" concern as Module 04's Book Library snapshot.
func (s *Stack[T]) Items() []T {
	out := make([]T, len(s.items))
	copy(out, s.items)
	return out
}

// Ordered is a CUSTOM CONSTRAINT — Stack[T any] itself never needs to
// compare elements, but a standalone function like MaxInStack does, so IT
// carries the narrower constraint instead of the whole Stack type.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~float32 | ~float64 | ~string
}

// MaxInStack works on any Stack[T] as long as T satisfies Ordered — this
// couldn't be a METHOD on Stack[T any] itself (a method can't narrow its
// receiver's type parameter to a different constraint), so it's a
// standalone function that takes a *Stack[T] instead.
func MaxInStack[T Ordered](s *Stack[T]) (T, bool) {
	var max T
	items := s.Items()
	if len(items) == 0 {
		return max, false
	}
	max = items[0]
	for _, item := range items[1:] {
		if item > max {
			max = item
		}
	}
	return max, true
}

// IsBalanced checks whether a string's brackets are properly nested, using
// Stack[rune] — a genuinely practical use of a generic stack, not just a
// toy demo. Every open bracket gets pushed; every close bracket must match
// whatever's currently on top.
func IsBalanced(s string) bool {
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	stack := NewStack[rune]()

	for _, ch := range s {
		switch ch {
		case '(', '[', '{':
			stack.Push(ch)
		case ')', ']', '}':
			top, ok := stack.Pop()
			if !ok || top != pairs[ch] {
				return false // closing bracket with nothing (or the wrong thing) open
			}
		}
	}
	return stack.IsEmpty() // balanced only if every opened bracket was closed
}

func main() {
	fmt.Println("=== Stack[int] ===")
	intStack := NewStack[int]()
	intStack.Push(1)
	intStack.Push(2)
	intStack.Push(3)
	for !intStack.IsEmpty() {
		val, _ := intStack.Pop()
		fmt.Println("popped:", val) // prints 3, 2, 1 — LIFO order
	}

	fmt.Println("\n=== MaxInStack — needs the Ordered constraint, not just `any` ===")
	scores := NewStack[int]()
	scores.Push(42)
	scores.Push(97)
	scores.Push(15)
	max, _ := MaxInStack(scores)
	fmt.Println("max score:", max)

	words := NewStack[string]()
	words.Push("banana")
	words.Push("apple")
	words.Push("cherry")
	maxWord, _ := MaxInStack(words) // Ordered includes ~string, so this works too
	fmt.Println("lexicographically largest:", maxWord)

	fmt.Println("\n=== IsBalanced — Stack[rune] solving a real problem ===")
	tests := []string{
		"({[]})",
		"([)]",
		"((()))",
		"(()",
		"",
	}
	for _, t := range tests {
		fmt.Printf("  %-10q balanced: %v\n", t, IsBalanced(t))
	}
}
