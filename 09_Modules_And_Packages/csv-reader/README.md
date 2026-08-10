# Project 2 — CSV Reader

```bash
cd csv-reader
go run main.go
```

Generates `data/employees.csv` on first run, reads and prints every row,
filters to Engineering, and writes `data/engineering.csv`.

## What's Demonstrated Here

- **Streaming with `encoding/csv`'s `Read()` loop** — `readEmployeesStreaming`
  calls `reader.Read()` one row at a time, checking `errors.Is(err, io.EOF)`
  to know when it's done, instead of `reader.ReadAll()` (which is simpler
  but loads every row into memory before you get any of them back).
- **Explicit buffering** — `os.Open`'s `*os.File` is wrapped in a
  `bufio.Reader` *before* being handed to `csv.NewReader`, so every
  underlying read benefits from buffering, not just the CSV-parsing layer.
- **Column lookup by name, not position** — `indexColumns` builds a
  `map[string]int` from the header row, so `row[colIndex["salary"]]` keeps
  working even if the CSV's column order ever changes, unlike a hardcoded
  `row[2]`.
- **The safe-write pattern again, plus explicit directory permissions** —
  `writeEmployeesCSV` creates its output directory with an explicit `0755`,
  then writes through a temp file, a `csv.Writer` *wrapping* a
  `bufio.Writer` (note the double-flush: `csvWriter.Flush()` empties csv's
  own small internal buffer into `bufWriter`, then `bufWriter.Flush()`
  actually pushes it all to disk), before the final `os.Rename`.

```
┌──────────────────────────────────────────────────────────┐
│   os.Open(path)  →  *os.File                                    │
│        │                                                            │
│        ▼                                                              │
│   bufio.NewReader(file)   ◀── buffers raw byte reads                    │
│        │                                                                    │
│        ▼                                                                       │
│   csv.NewReader(bufferedReader)   ◀── parses CSV structure ON TOP of              │
│        │                              the buffered byte stream                       │
│        ▼                                                                                 │
│   reader.Read()  one row at a time, until io.EOF                                            │
└──────────────────────────────────────────────────────────┘
```

## Case Study: The Double-Flush on the Writing Side

```go
csvWriter.Flush()             // 1. csv.Writer's OWN internal buffer -> bufWriter
if err := csvWriter.Error(); err != nil { return err }
if err := bufWriter.Flush(); err != nil { return err }  // 2. bufWriter -> the actual file
```

This looks redundant at first, but it's two *different* buffers being
flushed. `csv.Writer` keeps its own small internal state (partially because
CSV writing needs to handle quoting/escaping before bytes are final) and
writes to whatever `io.Writer` you gave it when you called `csv.NewWriter` —
here, that's `bufWriter`, **not** the file directly. Calling
`csvWriter.Flush()` pushes CSV's buffered output into `bufWriter`'s buffer —
but `bufWriter` itself still hasn't necessarily touched the disk yet, hence
the second `bufWriter.Flush()` right after. Skipping either one risks the
exact bug the guide warns about: data that looks written (no error!) but
never actually reached the file. Also worth noting: `csv.Writer.Flush()`
doesn't return an error directly — check `csvWriter.Error()` immediately
after calling it, which is exactly what this code does.

## Try It Yourself
- Change `filterByDepartment` to accept a `func(Employee) bool` predicate
  instead of a hardcoded department string — a callback (Module 03's
  higher-order functions), so filtering by salary range or name prefix needs
  no new function
- Add a `-column`/`-value` pair of flags (Module 00's `flag` package) so the
  filter is configurable from the command line instead of hardcoded to
  `"Engineering"`
- Read `data/employees.csv` with `csv.Reader.ReadAll()` instead, time both
  approaches on a much larger generated CSV (say, 500,000 rows), and compare
  memory behavior — a good way to see the streaming-vs-load-everything
  trade-off from the guide made concrete
