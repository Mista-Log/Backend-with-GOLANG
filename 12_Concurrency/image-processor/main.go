// Project 4: Image Processor
//
// A three-stage pipeline — load, resize (itself fan-out/fan-in across 3
// workers, since it's the CPU-bound stage), save — all connected by
// channels and running concurrently. Every stage respects context
// cancellation via select. A separate sync.Cond-based BoundedLog collects
// progress messages from all stages with real backpressure, contrasted
// against the channel-based approaches used everywhere else in this module.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// --- sync.Cond-based bounded log (backpressure, not just a channel) --------

// BoundedLog holds at most `capacity` unread entries — Push BLOCKS (via
// cond.Wait) once full, until the drainer goroutine frees space by reading
// some out. This is the guide's BoundedQueue example, applied to something
// this project actually uses: every pipeline stage logs through here, so a
// slow drainer naturally applies backpressure to fast stages.
type BoundedLog struct {
	mu       sync.Mutex
	cond     *sync.Cond
	entries  []string
	capacity int
}

func NewBoundedLog(capacity int) *BoundedLog {
	l := &BoundedLog{capacity: capacity}
	l.cond = sync.NewCond(&l.mu)
	return l
}

func (l *BoundedLog) Push(entry string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for len(l.entries) >= l.capacity { // `for`, not `if` — re-check after every wakeup
		l.cond.Wait()
	}
	l.entries = append(l.entries, entry)
	l.cond.Signal() // wake one goroutine that might be waiting for something to drain (not used here, but symmetrical)
}

// DrainAll removes and returns everything currently logged, then wakes any
// goroutines blocked in Push waiting for room to free up.
func (l *BoundedLog) DrainAll() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.entries
	l.entries = nil
	l.cond.Broadcast() // space freed — wake EVERY blocked Push, not just one
	return out
}

// --- Pipeline types -----------------------------------------------------

type Image struct {
	Path string
	Data []byte
}

type Result struct {
	Path       string
	OutputPath string
	Err        error
}

// --- Stage 1: load -----------------------------------------------------

func loadStage(ctx context.Context, log *BoundedLog, paths []string) <-chan Image {
	out := make(chan Image)
	go func() {
		defer close(out)
		for _, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				log.Push(fmt.Sprintf("LOAD FAILED %s: %v", p, err))
				continue
			}
			log.Push(fmt.Sprintf("loaded %s (%d bytes)", filepath.Base(p), len(data)))
			select {
			case out <- Image{Path: p, Data: data}:
			case <-ctx.Done(): // cancelled — stop feeding new work downstream
				return
			}
		}
	}()
	return out
}

// --- Stage 2: resize — itself a fan-out/fan-in across 3 workers, since   --
// --- it's the CPU-bound stage that benefits most from parallelism        --

func resizeStage(ctx context.Context, log *BoundedLog, in <-chan Image) <-chan Image {
	out := make(chan Image)
	const numResizeWorkers = 3
	var wg sync.WaitGroup

	for w := 0; w < numResizeWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for img := range in { // fan-OUT: all workers share the same `in`
				time.Sleep(time.Duration(20+rand.Intn(40)) * time.Millisecond) // simulate CPU work
				resized := Image{Path: img.Path, Data: img.Data[:len(img.Data)/2]} // "shrink" by half
				log.Push(fmt.Sprintf("resized %s on worker %d", filepath.Base(img.Path), workerID))
				select {
				case out <- resized: // fan-IN: all workers share the same `out`
				case <-ctx.Done():
					return
				}
			}
		}(w)
	}

	go func() {
		wg.Wait()  // wait for ALL resize workers to finish
		close(out) // THEN close — same fan-in pattern as the guide's fanIn helper
	}()

	return out
}

// --- Stage 3: save -----------------------------------------------------

func saveStage(ctx context.Context, log *BoundedLog, in <-chan Image, outDir string) <-chan Result {
	out := make(chan Result)
	go func() {
		defer close(out)
		for img := range in {
			outputPath := filepath.Join(outDir, "resized_"+filepath.Base(img.Path))
			err := os.WriteFile(outputPath, img.Data, 0644)
			if err == nil {
				log.Push(fmt.Sprintf("saved %s", filepath.Base(outputPath)))
			}
			result := Result{Path: img.Path, OutputPath: outputPath, Err: err}
			select {
			case out <- result:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// --- Setup + main -----------------------------------------------------

func generateSampleImages(dir string, count int) ([]string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	var paths []string
	for i := 1; i <= count; i++ {
		path := filepath.Join(dir, fmt.Sprintf("photo%02d.img", i))
		// Stand-in "image" data — real byte content, just not a real image
		// format; the pipeline only cares about moving and shrinking bytes.
		data := make([]byte, 200+rand.Intn(300))
		rand.Read(data)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func main() {
	const inputDir = "images/originals"
	const outputDir = "images/resized"

	paths, err := generateSampleImages(inputDir, 10)
	if err != nil {
		fmt.Println("Error generating sample images:", err)
		return
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Println("Error:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log := NewBoundedLog(4) // deliberately small capacity, so backpressure is visible

	// A dedicated drainer goroutine — the ONLY place log entries actually
	// get printed, freeing room for more Push calls each time it runs.
	var drainerWG sync.WaitGroup
	drainerWG.Add(1)
	go func() {
		defer drainerWG.Done()
		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, entry := range log.DrainAll() {
					fmt.Println(" ", entry)
				}
			case <-ctx.Done():
				for _, entry := range log.DrainAll() { // final flush
					fmt.Println(" ", entry)
				}
				return
			}
		}
	}()

	fmt.Printf("Processing %d images through the pipeline...\n\n", len(paths))

	// The pipeline itself: three stages, chained by channels, each stage
	// running concurrently with the others.
	loaded := loadStage(ctx, log, paths)
	resized := resizeStage(ctx, log, loaded)
	results := saveStage(ctx, log, resized, outputDir)

	successCount, failCount := 0, 0
	for result := range results {
		if result.Err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	cancel() // stop the drainer's select loop, triggering its final flush
	drainerWG.Wait()

	fmt.Printf("\nDone: %d succeeded, %d failed\n", successCount, failCount)
}
