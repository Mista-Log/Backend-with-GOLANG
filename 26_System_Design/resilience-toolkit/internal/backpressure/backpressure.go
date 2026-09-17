// Package backpressure demonstrates the guide's simplest real example: a
// bounded channel. Once full, submitting more work BLOCKS the caller —
// structural backpressure, no extra signaling code needed.
package backpressure

import "sync"

type Pool struct {
	jobs chan func()
	wg   sync.WaitGroup
}

// New starts numWorkers goroutines pulling from a channel buffered at
// queueSize — once queueSize jobs are queued and all workers are busy,
// Submit BLOCKS the caller, exactly the guide's "producer slows down"
// backpressure behavior.
func New(numWorkers, queueSize int) *Pool {
	p := &Pool{jobs: make(chan func(), queueSize)}
	for i := 0; i < numWorkers; i++ {
		go func() {
			for job := range p.jobs {
				job()
			}
		}()
	}
	return p
}

func (p *Pool) Submit(job func()) {
	p.wg.Add(1)
	p.jobs <- func() {
		defer p.wg.Done()
		job()
	}
}

func (p *Pool) Wait() { p.wg.Wait() }
func (p *Pool) Close() { close(p.jobs) }
