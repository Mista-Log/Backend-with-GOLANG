// Interview practice demo — run with: go run ./cmd/demo
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"interviewpractice/concurrency"
	"interviewpractice/dsa"
	"interviewpractice/internals"
)

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 66))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 66))
}

func main() {
	section("GO INTERNALS — the gotchas, proven")
	fmt.Println(internals.DemoNilInterface())
	fmt.Println()
	fmt.Println(internals.DemoSliceAliasing())
	fmt.Println()
	fmt.Println(internals.DemoNilVsEmpty())
	fmt.Println()
	fmt.Println(internals.DemoMapOrder())
	fmt.Println()
	fmt.Printf("loop-variable capture (Go 1.22+ per-iteration scoping): %v\n", internals.DemoLoopCapture())

	section("CONCURRENCY PROBLEMS")
	squared := concurrency.WorkerPool([]int{1, 2, 3, 4, 5, 6}, 3, func(n int) int { return n * n })
	fmt.Println("worker pool (3 workers, results reassembled in order):", squared)

	fmt.Println("alternating goroutines:", concurrency.Alternate(6))

	ch1 := make(chan int, 3)
	ch2 := make(chan int, 3)
	for i := 1; i <= 3; i++ {
		ch1 <- i
		ch2 <- i * 10
	}
	close(ch1)
	close(ch2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var merged []int
	for v := range concurrency.FanIn(ctx, ch1, ch2) {
		merged = append(merged, v)
	}
	fmt.Printf("fan-in merged %d values from 2 channels: %v\n", len(merged), merged)

	val, err := concurrency.WithTimeout(func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	}, 200*time.Millisecond)
	fmt.Println("timeout pattern (fast call):", val, err)

	_, err = concurrency.WithTimeout(func() (int, error) {
		time.Sleep(300 * time.Millisecond)
		return 42, nil
	}, 100*time.Millisecond)
	fmt.Println("timeout pattern (slow call):", err)

	section("DSA")
	fmt.Println("TwoSum([2,7,11,15], 9):            ", dsa.TwoSum([]int{2, 7, 11, 15}, 9))
	fmt.Println("LongestUniqueSubstring(abcabcbb):  ", dsa.LongestUniqueSubstring("abcabcbb"))
	fmt.Println("BinarySearch([1,3,5,7,9], 7):      ", dsa.BinarySearch([]int{1, 3, 5, 7, 9}, 7))
	fmt.Println("Permutations([1,2,3]):             ", dsa.Permutations([]int{1, 2, 3}))
	fmt.Println("GroupAnagrams:                     ", dsa.GroupAnagrams([]string{"eat", "tea", "tan", "ate", "nat", "bat"}))
	fmt.Println("MaxSubArray([-2,1,-3,4,-1,2,1,-5,4]):", dsa.MaxSubArray([]int{-2, 1, -3, 4, -1, 2, 1, -5, 4}))

	fmt.Println()
	fmt.Println("All solutions above have tests — run: go test ./...")
}
