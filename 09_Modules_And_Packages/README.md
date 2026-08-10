# Go for Beginners — Module 10: File Handling

## Contents

1. **[10-file-handling.md](./10-file-handling.md)** — Reading files
   (`os.ReadFile` vs. `os.Open`), writing files (`os.WriteFile`,
   `os.Create`, append flags), directories (`os.Mkdir`/`MkdirAll`,
   `os.ReadDir` vs. `filepath.WalkDir`), permissions (octal `FileMode`
   explained digit by digit), temp files and the safe-write
   (write-then-rename) pattern, streams (`io.Reader`/`io.Writer`, `io.Copy`),
   and buffers (`bufio`, and the "forgot to Flush" bug). Diagrams included
   throughout.

2. **[log-parser/](./log-parser)** — Generates sample `.log` files, streams
   each one line-by-line with `bufio.Scanner`, lists a directory with
   `os.ReadDir`, and writes an aggregated summary report through the full
   safe-write pattern (temp file → explicit `0644` permissions →
   `os.Rename`).

3. **[csv-reader/](./csv-reader)** — Generates a sample employee CSV, reads
   it row-by-row with `encoding/csv` wrapped around an explicit
   `bufio.Reader`, filters by department, and writes the result back out
   through the same safe-write pattern — with a case study on the
   double-flush needed when a `csv.Writer` wraps a `bufio.Writer`.

## Suggested Order

```
File handling guide ──▶ Log Parser ──▶ CSV Reader
                          (bufio.Scanner,     (encoding/csv streaming,
                           os.ReadDir)          nested buffered writers)
```

Both projects use the *same* safe-write pattern (temp file, explicit
permissions, atomic rename) deliberately, so it lands as a repeatable habit
for "write output that other things might be reading concurrently" — not
a one-off trick specific to either project.

## Quick Reference: Running Either Project

```bash
cd log-parser && go run main.go
cd csv-reader && go run main.go
```

Both generate their own sample input files on first run — nothing to set up
beforehand.

*Note: this module builds on Modules 00–08 — start there first if you
haven't already, especially Module 00's File Organizer (the origin of the
safe-write pattern) and Module 08's `errors.Is` (used here to detect
`io.EOF`).*
