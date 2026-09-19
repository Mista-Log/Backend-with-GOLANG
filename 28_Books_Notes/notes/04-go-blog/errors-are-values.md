# Errors are ordinary values, so ordinary programming techniques — not special syntax — are the right tools for reducing error-handling repetition

**Source**: "Errors are values", Rob Pike, Go Blog (2015)
**Why I read this**: The `if err != nil` repetition was bothering me and I
wanted to know whether idiomatic Go had an answer beyond "get used to it."

## The claim, in my words

Complaints about `if err != nil` verbosity usually come from treating
error handling as boilerplate you must write. Because errors are just
values, you can *program* with them — store one in a struct, check it
once at the end, and collapse many checks into one.

## Concrete example — the errWriter pattern from the post

```go
// BEFORE — a check after every single write
_, err = fd.Write(p0[a:b])
if err != nil { return err }
_, err = fd.Write(p1[c:d])
if err != nil { return err }
_, err = fd.Write(p2[e:f])
if err != nil { return err }

// AFTER — the error is STATE, checked once
type errWriter struct {
    w   io.Writer
    err error
}
func (ew *errWriter) write(buf []byte) {
    if ew.err != nil { return }   // once failed, all subsequent writes no-op
    _, ew.err = ew.w.Write(buf)
}

ew := &errWriter{w: fd}
ew.write(p0[a:b])
ew.write(p1[c:d])
ew.write(p2[e:f])
if ew.err != nil { return ew.err }   // ONE check
```

**What actually happened when I applied it**: this is exactly how
`bufio.Writer` and `strings.Builder` behave internally — errors are
sticky, checked at `Flush()`. Recognizing the pattern in stdlib made it
click.

## When this applies / doesn't

**Applies** when many operations share a failure mode and you'd abort on
any of them anyway — sequential writes, a series of parses.

**Doesn't apply** when each error needs *different* handling, or needs
context added per-step. There, explicit `if err != nil` with
`fmt.Errorf("...: %w", err)` (Module 08) is correct and clearer.

**The deeper point**: don't reach for this to reduce line count. Reach for
it when the error-checking genuinely *is* repetitive in a way that
obscures the actual logic.

## Links

- Module 08 (error wrapping, sentinel errors, errors.Is/As)
- Module 05 (bufio.Writer's sticky error behavior)
