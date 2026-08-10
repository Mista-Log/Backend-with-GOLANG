# 10. File Handling

Everything in this module builds on `os`, `bufio`, and `io` — the same
packages quietly underpinning several projects from earlier modules (Module
00's File Organizer, Module 06's Storage Drivers). Now it's time to cover
them properly.

---

## 1. Reading Files

**Small files — read the whole thing at once:**
```go
data, err := os.ReadFile("config.yaml")
if err != nil {
	// handle it — os.ReadFile wraps the underlying error with the path
}
fmt.Println(string(data))
```

**Larger files, or when you want more control — open, then read:**
```go
f, err := os.Open("data.txt") // opens READ-ONLY
if err != nil {
	return err
}
defer f.Close() // Module 02's defer, doing exactly what it's for

buf := make([]byte, 1024)
n, err := f.Read(buf) // reads UP TO len(buf) bytes, returns how many it actually got
```

```
┌────────────────────────────────────────────────────┐
│   os.ReadFile(path)   →  entire file, all at once,        │
│                            as a []byte — simplest,             │
│                            but loads everything into            │
│                            memory upfront                          │
│                                                                        │
│   os.Open(path)       →  an *os.File you read from             │
│                            incrementally — the right                │
│                            choice for large files, or                 │
│                            when you want to process a                   │
│                            file as a STREAM (see below)                    │
└────────────────────────────────────────────────────┘
```

---

## 2. Writing Files

**Small writes:**
```go
err := os.WriteFile("output.txt", []byte("hello\n"), 0644) // 0644 = permissions, see below
```

**Larger or incremental writes:**
```go
f, err := os.Create("output.txt") // creates (or TRUNCATES an existing file!), write-only
if err != nil {
	return err
}
defer f.Close()

f.WriteString("line one\n")
f.WriteString("line two\n")
```

**Appending instead of overwriting** needs explicit flags:
```go
f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
```

```
┌────────────────────────────────────────────────────┐
│   os.WriteFile(...)  →  write it all at once                │
│   os.Create(...)      →  TRUNCATES if the file exists,          │
│                            starts empty, write-only                 │
│   os.OpenFile(..., flags, perm)  →  full control: append,             │
│                            create-if-missing, read/write,               │
│                            exclusive-create, etc.                          │
└────────────────────────────────────────────────────┘
```

---

## 3. Directories

```go
os.Mkdir("reports", 0755)          // one directory — fails if a parent is missing
os.MkdirAll("reports/2026/june", 0755) // creates every missing parent along the way

entries, err := os.ReadDir("reports") // lists immediate contents (files AND dirs)
for _, e := range entries {
	fmt.Println(e.Name(), e.IsDir())
}

// Walk an entire tree recursively:
filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
})
```

`os.ReadDir` (non-recursive, one level) vs. `filepath.WalkDir` (recursive,
visits every file and subdirectory) is the same distinction Module 00's File
Organizer called out in its "try it yourself" section — worth remembering
now that you have both tools in hand.

---

## 4. Permissions

Go represents Unix-style file permissions as an `os.FileMode`, almost always
written in octal (`0644`, `0755`, ...) — each digit covers owner, group, and
everyone else.

```
┌────────────────────────────────────────────────────┐
│        0   6    4    4                                    │
│        │   │    │    │                                       │
│    (unused│    group  others                                    │
│     here) owner                                                    │
│                                                                        │
│   Each digit is a sum of:  read(4) + write(2) + execute(1)              │
│                                                                              │
│   6 = 4+2       → read + write                                                │
│   4 = 4          → read only                                                    │
│   7 = 4+2+1        → read + write + execute                                        │
│                                                                                          │
│   0644  →  owner: read+write,  group: read,  others: read                                 │
│   0755  →  owner: read+write+execute,  group: read+execute,  others: read+execute              │
└────────────────────────────────────────────────────┘
```

`0644` is the standard default for regular files (readable by everyone,
writable only by the owner); `0755` is standard for directories and
executables (need the execute bit to be enterable/runnable at all). Check or
change permissions on an existing file:

```go
info, _ := os.Stat("script.sh")
fmt.Println(info.Mode())      // e.g. -rw-r--r--
os.Chmod("script.sh", 0755)     // make it executable
```

---

## 5. Temp Files

For scratch work, or for the **safe-write pattern** (write to a temp file,
then atomically rename it into place so readers never see a half-written
file), use `os.CreateTemp`:

```go
tmp, err := os.CreateTemp("", "myapp-*.tmp") // "" = system temp dir, * = random suffix
if err != nil {
	return err
}
defer os.Remove(tmp.Name()) // clean up if something below fails before the rename

tmp.WriteString("partial data...")
tmp.Close()

os.Rename(tmp.Name(), "final-output.txt") // Module 00's File Organizer used
                                            // exactly this call, for exactly
                                            // this atomicity guarantee
```

```
┌────────────────────────────────────────────────────────┐
│   write to a TEMP file  ──▶  fully written & closed  ──▶  os.Rename    │
│                                                              into the       │
│                                                              REAL path        │
│                                                                                   │
│   A reader checking the real path at ANY point during this either sees            │
│   the OLD complete file, or the NEW complete file — never a half-written             │
│   one, because os.Rename is atomic (same filesystem/volume — see Module               │
│   00's File Organizer case study for the full explanation).                              │
└────────────────────────────────────────────────────────┘
```

---

## 6. Streams

A "stream" in Go isn't a specific type — it's the *pattern* of processing
data incrementally, piece by piece, instead of loading it all into memory
first. `*os.File` is a stream; so is an HTTP response body, a network
connection, or `os.Stdin`. The unifying interfaces (from Module 06) are
`io.Reader` and `io.Writer`:

```go
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
```

Because `*os.File` satisfies both, and so does almost everything else that
moves bytes around in Go, functions written against `io.Reader`/`io.Writer`
work identically whether the data is coming from a file, the network, or an
in-memory buffer:

```go
func countBytes(r io.Reader) (int, error) {
	buf := make([]byte, 4096)
	total := 0
	for {
		n, err := r.Read(buf)
		total += n
		if err == io.EOF { // io.EOF signals a clean end of stream, not a failure
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

countBytes(myFile)          // works
countBytes(strings.NewReader("hello")) // ALSO works — same function, different source
```

`io.Copy(dst, src)` is the standard shortcut for "stream everything from one
place to another" without manually looping:
```go
io.Copy(outFile, inFile) // efficient, chunked copy — no need to hand-roll the loop above
```

---

## 7. Buffers

Reading or writing one byte (or one line) at a time directly against a file
is slow — every `Read`/`Write` call is a real system call, and system calls
are expensive relative to in-memory work. `bufio` wraps a reader or writer
with an in-memory buffer that batches those calls.

```go
f, _ := os.Open("large-file.txt")
defer f.Close()

scanner := bufio.NewScanner(f)
for scanner.Scan() {
	line := scanner.Text() // one line at a time, buffered internally —
	fmt.Println(line)       // far fewer actual syscalls than reading byte-by-byte
}

writer := bufio.NewWriter(outFile)
defer writer.Flush() // IMPORTANT — buffered writes aren't on disk until flushed!
for _, line := range lines {
	writer.WriteString(line + "\n")
}
```

```
┌────────────────────────────────────────────────────────┐
│   Unbuffered:  Write("a") → syscall.  Write("b") → syscall.    │
│                Write("c") → syscall.  ... (SLOW: one syscall        │
│                per write)                                              │
│                                                                            │
│   Buffered:    Write("a"), Write("b"), Write("c") all land in an           │
│                in-memory buffer FIRST — one real syscall happens             │
│                only when the buffer fills up, or you call Flush()               │
└────────────────────────────────────────────────────────┘
```

**The single most common bug with buffered writers**: forgetting to
`Flush()` (or the flush never running because of an early return) — the
data sits in memory, looking successfully "written," until the buffer is
explicitly flushed or closed. This is exactly why `defer writer.Flush()`
appears immediately after creating a `bufio.Writer`, the same defensive
habit as `defer f.Close()`.

---

Onto the projects — Log Parser leans on `bufio.Scanner` for line-by-line
streaming and directory listing for batch processing; CSV Reader leans on
`encoding/csv`, the temp-file-then-rename safe-write pattern, and explicit
permission handling for its output directory.
