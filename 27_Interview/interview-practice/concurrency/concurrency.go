// Package concurrency holds the classic Go concurrency interview
// problems — the ones that come up repeatedly, with the reasoning that
// makes each answer defensible out loud, not just correct.
package concurrency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// --- 1. Worker Pool -------------------------------------------------
// "Process N jobs with at most K workers running concurrently."

func WorkerPool(jobs []int, numWorkers int, process func(int) int) []int {
	jobCh := make(chan int, len(jobs))
	type result struct{ index, value int }
	resultCh := make(chan result, len(jobs))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobCh {
				resultCh <- result{index: idx, value: process(jobs[idx])}
			}
		}()
	}

	for i := range jobs {
		jobCh <- i
	}
	close(jobCh) // signals workers: no more jobs

	wg.Wait()
	close(resultCh)

	// Results arrive out of order — reassemble by index, since the
	// interviewer will almost always ask "does order matter?" next.
	out := make([]int, len(jobs))
	for r := range resultCh {
		out[r.index] = r.value
	}
	return out
}

// --- 2. Alternating Goroutines ---------------------------------------
// "Two goroutines print odd and even numbers in strict alternation."
// The key insight: use UNBUFFERED channels as a handoff baton — each
// goroutine blocks until the other hands control back.

func Alternate(n int) []string {
	var output []string
	var mu sync.Mutex

	oddTurn := make(chan struct{})
	evenTurn := make(chan struct{})
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // odd printer
		defer wg.Done()
		for i := 1; i <= n; i += 2 {
			<-oddTurn
			mu.Lock()
			output = append(output, fmt.Sprintf("odd:%d", i))
			mu.Unlock()
			if i+1 > n {
				close(done)
				return
			}
			evenTurn <- struct{}{}
		}
	}()

	go func() { // even printer
		defer wg.Done()
		for i := 2; i <= n; i += 2 {
			<-evenTurn
			mu.Lock()
			output = append(output, fmt.Sprintf("even:%d", i))
			mu.Unlock()
			if i+1 > n {
				close(done)
				return
			}
			oddTurn <- struct{}{}
		}
	}()

	oddTurn <- struct{}{} // start the chain
	<-done
	wg.Wait()
	return output
}

// --- 3. Fan-In with Context Cancellation -----------------------------
// "Merge several channels into one, stopping cleanly on cancellation."

func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for {
				select {
				case v, ok := <-c:
					if !ok {
						return
					}
					select {
					case out <- v:
					case <-ctx.Done(): // don't block forever if nobody's reading
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out) // only AFTER every source is drained
	}()
	return out
}

// --- 4. Timeout Pattern ----------------------------------------------
// "Call a function, but give up after a timeout."

func WithTimeout(fn func() (int, error), timeout time.Duration) (int, error) {
	type res struct {
		val int
		err error
	}
	ch := make(chan res, 1) // BUFFERED 1 — so the goroutine never leaks if we time out

	go func() {
		v, err := fn()
		ch <- res{v, err}
	}()

	select {
	case r := <-ch:
		return r.val, r.err
	case <-time.After(timeout):
		return 0, fmt.Errorf("operation timed out after %v", timeout)
	}
}
