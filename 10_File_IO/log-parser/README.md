# Project 1 — Log Parser

```bash
cd log-parser
go run main.go
```

This generates a `logs/` folder with two sample `.log` files on first run (so
there's nothing to set up), parses both, prints every `ERROR` line as it's
found, and writes `logs/summary.txt`.

## What's Demonstrated Here

- **Streaming with `bufio.Scanner`** — `ParseFile` never loads a whole log
  file into memory; it reads and processes one line at a time. This is the
  difference that matters once "sample log file" becomes "years of
  production logs."
- **`os.ReadDir` for batch processing** — `findLogFiles` lists every `.log`
  file in a directory (non-recursively — see the guide's note on
  `os.ReadDir` vs. `filepath.WalkDir` if you want to search subdirectories
  too).
- **The safe-write pattern, for real this time** — `writeSummaryReport`
  writes to a temp file via `os.CreateTemp`, flushes and closes it, sets its
  final permissions explicitly with `os.Chmod`, and only then `os.Rename`s
  it into place. If anything fails partway through, `logs/summary.txt`
  either doesn't exist yet or holds the *previous* successful report — never
  a half-written one.
- **Explicit permissions (`0644`)** — set deliberately on the report,
  independent of whatever mode `os.CreateTemp` happened to create the
  scratch file with.

```
┌──────────────────────────────────────────────────────────┐
│   ParseFile("logs/app.log")                                      │
│        │                                                            │
│        ▼                                                              │
│   os.Open  →  bufio.NewScanner  →  Scan() one line at a time            │
│        │                              │                                    │
│        │                              ▼                                       │
│        │                        ParseLine(line) → LogEntry or error              │
│        ▼                                                                            │
│   defer f.Close()  (runs when ParseFile returns, either way)                          │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why the Report Isn't Just `os.WriteFile(path, data, 0644)`

For a report this small, `os.WriteFile` directly would work fine in
practice — but imagine this report took 30 seconds to generate (a much
larger log set, or a slower aggregation step) and the program crashed, or
was killed, halfway through writing it. With a direct write, anyone reading
`summary.txt` at that moment — a dashboard polling the file, a teammate
tailing it — would see a **truncated, half-written report** and have no way
to tell it was incomplete. The temp-file-then-rename pattern (first
introduced in Module 00's File Organizer, for a different reason — atomic
moves) solves this here too: `os.Rename` is atomic on the same filesystem,
so the report at `logs/summary.txt` is always either the complete previous
version or the complete new version, with no window where it's visible in a
broken, partial state.

## Try It Yourself
- Add a `-dir` flag (Module 00's `flag` package) so this can parse any
  directory of logs, not just the hardcoded `logs/` folder
- Switch `findLogFiles` to `filepath.WalkDir` so it also picks up `.log`
  files in subdirectories, and compare how much the function body has to
  change
- Add a time-range filter (`-since`, `-until`) using `LogEntry.Timestamp`,
  and only include matching entries in both the console `ERROR` printout and
  the summary report
