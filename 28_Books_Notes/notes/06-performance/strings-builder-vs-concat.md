# strings.Builder turns O(n^2) string concatenation into amortized O(n) — the gap widens with input size

**Source**: Module 23's optimize-rest-api benchmarks + `strings/builder.go`
**Why I read this**: Wanted actual numbers rather than "everyone says
`+=` in a loop is slow."

## The claim, in my words

`s += x` allocates a brand-new string and copies everything accumulated
so far, every iteration — so building n pieces copies roughly n^2/2 bytes
total. `strings.Builder` appends into one growing buffer (like `append`
on a `[]byte`), so total copying is amortized linear.

## The benchmark

```go
func BenchmarkConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 1000; j++ { s += "row\n" }
        _ = s
    }
}

func BenchmarkBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        sb.Grow(4000)                    // pre-size when you know it
        for j := 0; j < 1000; j++ { sb.WriteString("row\n") }
        _ = sb.String()
    }
}
```

**Results** (record YOUR numbers here — these are illustrative, from
Module 23's README, not measured on this machine):

```
BenchmarkConcat-8      ~2,847,193 ns/op   1,048,912 B/op   3011 allocs/op
BenchmarkBuilder-8       ~182,004 ns/op      49,536 B/op     12 allocs/op
```

## When this applies / doesn't

**Applies** to loops building strings from many pieces — reports, CSV,
generated SQL, log lines.

**Doesn't apply** to a handful of concatenations. `a + b + c` in one
expression is fine and clearer; the compiler handles it in one allocation.
Don't reach for `Builder` for two pieces.

**Also worth knowing**: `strings.Join(slice, sep)` is usually *better*
than a Builder loop when you already have a slice — it pre-computes the
exact size in one pass. Builder wins when you're generating pieces
incrementally and don't have them all up front.

## Links

- Module 23 (the full optimize-rest-api project, profiling)
- Module 04 (slice growth — Builder uses the same amortized strategy)
