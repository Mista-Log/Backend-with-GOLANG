// Project 3: File Organizer
//
// Scans a directory and moves each file into a subfolder named after its
// extension (e.g. photo.jpg -> Images/photo.jpg, notes.pdf -> Documents/notes.pdf).
//
// Usage:
//   go run main.go -dir ./messy-folder            (dry run by default — prints a plan)
//   go run main.go -dir ./messy-folder -apply      (actually moves the files)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// categoryFor maps a file extension to a human-friendly category/folder name.
// A map keeps this easy to extend without touching any logic below.
var categoryFor = map[string]string{
	".jpg": "Images", ".jpeg": "Images", ".png": "Images", ".gif": "Images",
	".pdf": "Documents", ".doc": "Documents", ".docx": "Documents", ".txt": "Documents",
	".mp3": "Audio", ".wav": "Audio",
	".mp4": "Video", ".mov": "Video",
	".zip": "Archives", ".tar": "Archives", ".gz": "Archives",
	".go": "Code", ".py": "Code", ".js": "Code", ".ts": "Code",
}

const defaultCategory = "Other"

// plan is one proposed move: a file going from Source to Dest.
type plan struct {
	source string
	dest   string
}

// buildPlan reads the directory (non-recursively — only top-level files) and
// works out where each file *would* go, without touching the filesystem yet.
// Separating "decide what to do" from "actually do it" makes the whole program
// testable and lets us safely offer a dry-run mode.
func buildPlan(dir string) ([]plan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var plans []plan
	for _, entry := range entries {
		if entry.IsDir() {
			continue // don't reorganize folders we already created, or existing ones
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		category, known := categoryFor[ext]
		if !known {
			category = defaultCategory
		}

		src := filepath.Join(dir, entry.Name())
		dst := filepath.Join(dir, category, entry.Name())
		plans = append(plans, plan{source: src, dest: dst})
	}

	// Sort for stable, predictable output (map/dir iteration order isn't guaranteed).
	sort.Slice(plans, func(i, j int) bool { return plans[i].source < plans[j].source })
	return plans, nil
}

// apply actually performs the moves, creating destination folders as needed.
func apply(plans []plan) error {
	for _, p := range plans {
		destDir := filepath.Dir(p.dest)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("creating folder %s: %w", destDir, err)
		}
		if err := os.Rename(p.source, p.dest); err != nil {
			return fmt.Errorf("moving %s: %w", p.source, err)
		}
	}
	return nil
}

func main() {
	dir := flag.String("dir", ".", "directory to organize")
	doApply := flag.Bool("apply", false, "actually move files (default: dry run only)")
	flag.Parse()

	plans, err := buildPlan(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if len(plans) == 0 {
		fmt.Println("Nothing to organize — no top-level files found in", *dir)
		return
	}

	fmt.Printf("Found %d file(s) in %s\n\n", len(plans), *dir)
	for _, p := range plans {
		fmt.Printf("  %s  ->  %s\n", filepath.Base(p.source), p.dest)
	}

	if !*doApply {
		fmt.Println("\nDry run only — nothing was moved. Re-run with -apply to execute this plan.")
		return
	}

	if err := apply(plans); err != nil {
		fmt.Fprintln(os.Stderr, "\nError while organizing:", err)
		os.Exit(1)
	}
	fmt.Println("\nDone — all files organized.")
}
