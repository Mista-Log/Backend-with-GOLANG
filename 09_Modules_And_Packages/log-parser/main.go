// Project 1: Log Parser
//
// Generates a handful of sample log files, then parses every *.log file in
// a directory using bufio.Scanner (streaming, not loading whole files into
// memory), aggregates level counts across all of them, and writes a summary
// report using the temp-file-then-rename safe-write pattern.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// ParseLine expects lines shaped like:
//   2026-08-10 14:23:01 [ERROR] connection to database lost
func ParseLine(line string) (*LogEntry, error) {
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 4 {
		return nil, fmt.Errorf("malformed line: %q", line)
	}
	ts, err := time.Parse("2006-01-02 15:04:05", parts[0]+" "+parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad timestamp in line %q: %w", line, err)
	}
	level := strings.Trim(parts[2], "[]")
	message := parts[3]
	return &LogEntry{Timestamp: ts, Level: level, Message: message}, nil
}

// ParseFile STREAMS the file with bufio.Scanner — even a multi-gigabyte log
// file only ever holds one line in memory at a time here, unlike
// os.ReadFile, which would load the entire thing upfront.
func ParseFile(path string) ([]LogEntry, []error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []error{err}
	}
	defer f.Close()

	var entries []LogEntry
	var parseErrors []error

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, err := ParseLine(line)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		entries = append(entries, *entry)
	}
	if err := scanner.Err(); err != nil {
		parseErrors = append(parseErrors, fmt.Errorf("scanning %s: %w", path, err))
	}
	return entries, parseErrors
}

// findLogFiles lists *.log files directly inside dir (non-recursive, via
// os.ReadDir — the Module 10 guide's distinction from filepath.WalkDir).
func findLogFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// generateSampleLogs creates a small "logs/" directory with a couple of
// sample files, so this project runs out of the box with no setup needed.
func generateSampleLogs(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	app := `2026-08-10 09:00:01 [INFO] server started on :8080
2026-08-10 09:01:15 [INFO] handled GET /health
2026-08-10 09:03:42 [WARN] slow query took 812ms
2026-08-10 09:05:10 [ERROR] connection to database lost
2026-08-10 09:05:11 [INFO] retrying database connection
2026-08-10 09:05:12 [INFO] database connection restored
2026-08-10 09:12:30 [ERROR] failed to process payment for order 4821
`
	worker := `2026-08-10 08:55:00 [INFO] worker pool starting, 4 workers
2026-08-10 09:02:18 [WARN] job queue depth exceeds 100
2026-08-10 09:04:05 [ERROR] job 993 failed: timeout
2026-08-10 09:04:06 [INFO] job 993 requeued
2026-08-10 09:20:00 [INFO] worker pool shutting down cleanly
`
	if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte(app), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "worker.log"), []byte(worker), 0644)
}

// writeSummaryReport uses the SAFE-WRITE pattern from the guide: write to a
// temp file first, only os.Rename into the final path once writing
// succeeds completely — a partially-written report never becomes visible
// at the real path.
func writeSummaryReport(path string, counts map[string]int, totalErrors int) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "summary-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op if the rename below succeeds first

	// bufio.Writer here is arguably overkill for a report this small, but
	// it's the right habit for any report that could grow — remember the
	// guide's warning: Flush() before Close(), or buffered writes vanish.
	w := bufio.NewWriter(tmp)
	fmt.Fprintln(w, "=== Log Summary Report ===")
	fmt.Fprintf(w, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	var levels []string
	for level := range counts {
		levels = append(levels, level)
	}
	sort.Strings(levels)
	for _, level := range levels {
		fmt.Fprintf(w, "%-8s %d\n", level+":", counts[level])
	}
	fmt.Fprintf(w, "\nUnparseable lines: %d\n", totalErrors)

	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Explicit permission, per the guide's section 4: a report is a
	// regular, non-executable file, so 0644 (owner read/write, everyone
	// else read-only) is the right mode, regardless of the temp file's
	// own (more restrictive, OS-chosen) default mode.
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func main() {
	const logsDir = "logs"
	const reportPath = "logs/summary.txt"

	if err := generateSampleLogs(logsDir); err != nil {
		fmt.Println("Error generating sample logs:", err)
		return
	}

	logFiles, err := findLogFiles(logsDir)
	if err != nil {
		fmt.Println("Error listing log files:", err)
		return
	}
	fmt.Printf("Found %d log file(s): %v\n\n", len(logFiles), logFiles)

	counts := make(map[string]int)
	totalErrors := 0

	for _, path := range logFiles {
		entries, parseErrors := ParseFile(path)
		fmt.Printf("--- %s (%d entries, %d parse errors) ---\n", path, len(entries), len(parseErrors))
		for _, e := range entries {
			counts[e.Level]++
			if e.Level == "ERROR" {
				fmt.Printf("  %s ERROR: %s\n", e.Timestamp.Format("15:04:05"), e.Message)
			}
		}
		totalErrors += len(parseErrors)
	}

	fmt.Println("\n=== Level counts across all files ===")
	for level, count := range counts {
		fmt.Printf("  %-8s %d\n", level+":", count)
	}

	if err := writeSummaryReport(reportPath, counts, totalErrors); err != nil {
		fmt.Println("Error writing report:", err)
		return
	}
	fmt.Printf("\nReport written to %s\n", reportPath)
}
