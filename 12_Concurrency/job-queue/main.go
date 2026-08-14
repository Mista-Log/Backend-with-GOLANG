// Project 2: Job Queue
//
// A worker pool processing arbitrary jobs, tracking each job's status in a
// Mutex-protected map (deliberately NOT sync.Map — see the README's case
// study on why a plain map + Mutex is the better fit here), with
// select-based cancellation via context and a clean, WaitGroup-coordinated
// shutdown.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusRunning JobStatus = "running"
	StatusDone    JobStatus = "done"
	StatusFailed  JobStatus = "failed"
)

type Job struct {
	ID   int
	Task func() (string, error)
}

type JobResult struct {
	ID     int
	Output string
	Err    error
}

type JobQueue struct {
	jobs    chan Job
	results chan JobResult
	wg      sync.WaitGroup

	statusMu sync.Mutex // protects status — see README for why NOT sync.Map here
	status   map[int]JobStatus
}

func NewJobQueue(ctx context.Context, numWorkers int) *JobQueue {
	q := &JobQueue{
		jobs:    make(chan Job, 100),
		results: make(chan JobResult, 100),
		status:  make(map[int]JobStatus),
	}
	for i := 1; i <= numWorkers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
	return q
}

// worker uses SELECT to wait on EITHER the next job OR context cancellation
// — whichever happens first. This is what lets Shutdown-via-cancel stop
// workers immediately, even mid-queue, instead of forcing them to drain
// every remaining job first.
func (q *JobQueue) worker(ctx context.Context, id int) {
	defer q.wg.Done()
	for {
		select {
		case job, ok := <-q.jobs:
			if !ok { // jobs channel closed and fully drained
				return
			}
			q.setStatus(job.ID, StatusRunning)
			output, err := job.Task()
			if err != nil {
				q.setStatus(job.ID, StatusFailed)
			} else {
				q.setStatus(job.ID, StatusDone)
			}
			q.results <- JobResult{ID: job.ID, Output: output, Err: err}

		case <-ctx.Done():
			return // cancelled — stop picking up NEW jobs immediately
		}
	}
}

func (q *JobQueue) setStatus(id int, status JobStatus) {
	q.statusMu.Lock()
	defer q.statusMu.Unlock()
	q.status[id] = status
}

func (q *JobQueue) Status(id int) JobStatus {
	q.statusMu.Lock()
	defer q.statusMu.Unlock()
	return q.status[id]
}

// Submit records the job as Pending BEFORE sending it to the channel, so
// Status() never returns "unknown" for a job that's already been accepted.
func (q *JobQueue) Submit(job Job) {
	q.setStatus(job.ID, StatusPending)
	q.jobs <- job
}

// Shutdown closes the jobs channel (no more NEW jobs accepted), waits for
// every in-flight job to actually finish, then closes results so any
// `range` over it terminates cleanly.
func (q *JobQueue) Shutdown() {
	close(q.jobs)
	q.wg.Wait()
	close(q.results)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := NewJobQueue(ctx, 4)

	// Collect results CONCURRENTLY with submission — if this ran after all
	// Submits instead, the results channel (buffered at 100) could still
	// work for this demo's size, but it wouldn't scale to more jobs than
	// the buffer holds. Reading concurrently is the general, correct shape.
	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for result := range queue.results {
			if result.Err != nil {
				fmt.Printf("  job %-3d FAILED: %v\n", result.ID, result.Err)
			} else {
				fmt.Printf("  job %-3d done:   %s\n", result.ID, result.Output)
			}
		}
	}()

	fmt.Println("Submitting 12 jobs to a 4-worker pool...")
	for i := 1; i <= 12; i++ {
		id := i
		queue.Submit(Job{
			ID: id,
			Task: func() (string, error) {
				time.Sleep(time.Duration(20+rand.Intn(60)) * time.Millisecond)
				if id%5 == 0 { // every 5th job deliberately fails
					return "", fmt.Errorf("simulated failure processing job %d", id)
				}
				return fmt.Sprintf("processed payload for job %d", id), nil
			},
		})
	}

	// A brief window where jobs are IN FLIGHT — status queries here show a
	// realistic mix of pending/running/done, proving the status map is
	// genuinely live, not just set-once-at-submit-time.
	time.Sleep(30 * time.Millisecond)
	fmt.Println("\nMid-flight status snapshot:")
	for id := 1; id <= 12; id++ {
		fmt.Printf("  job %-3d -> %s\n", id, queue.Status(id))
	}

	fmt.Println("\nShutting down (waiting for all in-flight jobs to finish)...")
	queue.Shutdown()
	collectorWG.Wait()

	fmt.Println("\nFinal status of every job:")
	for id := 1; id <= 12; id++ {
		fmt.Printf("  job %-3d -> %s\n", id, queue.Status(id))
	}
}
