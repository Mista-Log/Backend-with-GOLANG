// Package internals PROVES the interview gotchas from
// go-internals-reference.md actually behave as claimed — run
// cmd/demo to see each one demonstrated rather than just asserted.
package internals

import "fmt"

// --- The nil interface trap -------------------------------------------

type MyError struct{ Msg string }

func (e *MyError) Error() string { return e.Msg }

// BrokenNilCheck returns a nil *MyError as an error — the interface is
// NOT nil, because it carries a type.
func BrokenNilCheck() error {
	var e *MyError // nil pointer
	return e       // wrapped in an interface carrying type *MyError
}

func CorrectNilCheck() error {
	return nil // an actually-nil interface
}

func DemoNilInterface() string {
	broken := BrokenNilCheck()
	correct := CorrectNilCheck()
	return fmt.Sprintf(
		"BrokenNilCheck() == nil: %v  (surprising!)\nCorrectNilCheck() == nil: %v",
		broken == nil, correct == nil,
	)
}

// --- The slice aliasing trap ------------------------------------------

func DemoSliceAliasing() string {
	a := []int{1, 2, 3, 4, 5}
	b := a[1:3] // len 2, cap 4 — shares a's backing array
	b = append(b, 99)

	safe := []int{1, 2, 3, 4, 5}
	c := safe[1:3:3] // FULL slice expression caps it — append must reallocate
	c = append(c, 99)

	return fmt.Sprintf(
		"after append to a[1:3]:    a = %v  (a[3] was OVERWRITTEN)\n"+
			"after append to a[1:3:3]:  a = %v  (unchanged — capped)",
		a, safe,
	)
}

// --- nil vs empty slice ------------------------------------------------

func DemoNilVsEmpty() string {
	var nilSlice []int
	emptySlice := []int{}
	return fmt.Sprintf(
		"var a []int    → a == nil: %v, len: %d\n"+
			"b := []int{}   → b == nil: %v, len: %d\n"+
			"(both are safe to append/range; they differ in JSON: null vs [])",
		nilSlice == nil, len(nilSlice), emptySlice == nil, len(emptySlice),
	)
}

// --- Map iteration randomization ---------------------------------------

func DemoMapOrder() string {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}
	var runs []string
	for i := 0; i < 3; i++ {
		var order string
		for k := range m {
			order += k
		}
		runs = append(runs, order)
	}
	return fmt.Sprintf("three separate range passes over the SAME map: %v\n"+
		"(order differs — randomized deliberately, per the spec)", runs)
}

// --- Closure capture in loops (Go 1.22+ behavior) -----------------------

func DemoLoopCapture() []int {
	var results []int
	var fns []func()
	for i := 0; i < 3; i++ {
		fns = append(fns, func() { results = append(results, i) })
	}
	for _, fn := range fns {
		fn()
	}
	// Go 1.22+: per-iteration scoping → [0 1 2]
	// Go 1.21 and earlier: shared variable → [3 3 3]
	return results
}
