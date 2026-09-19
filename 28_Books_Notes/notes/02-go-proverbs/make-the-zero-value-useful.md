# A type whose zero value works without initialization removes an entire class of nil-check from every caller

**Source**: Go Proverbs (Rob Pike, 2015) + Effective Go
**Why I read this**: I kept writing `if x.field == nil { x.field = make(...) }`
guards at the top of methods and wondered if that was avoidable.

## The claim, in my words

If a type is designed so its zero value is immediately usable, callers
never need a constructor, and the type never needs internal
"am I initialized?" checks. `sync.Mutex`, `bytes.Buffer`, and
`strings.Builder` all work this way — `var mu sync.Mutex` is ready to
use, no `NewMutex()` anywhere.

## Concrete example

```go
// BAD — every method must guard, and callers must remember the constructor
type BadCounter struct{ counts map[string]int }
func (c *BadCounter) Inc(k string) {
    if c.counts == nil { c.counts = map[string]int{} } // guard in EVERY method
    c.counts[k]++
}

// GOOD — zero value is useful, no constructor needed at all
type GoodCounter struct {
    mu     sync.Mutex   // zero value: unlocked, ready
    counts sync.Map     // zero value: empty, ready
}
func (c *GoodCounter) Inc(k string) {
    v, _ := c.counts.LoadOrStore(k, new(atomic.Int64))
    v.(*atomic.Int64).Add(1)
}

var c GoodCounter  // no constructor call — just works
c.Inc("hits")
```

**What actually happened when I ran it**: `var c GoodCounter; c.Inc("x")`
works with no initialization. The `BadCounter` version panics on
`c.counts[k]++` without its guard (assignment to entry in nil map).

## When this applies / doesn't

**Applies** when the empty/zero state is semantically meaningful ("no
entries yet," "unlocked," "empty buffer").

**Doesn't apply** when the type genuinely requires configuration to be
valid — a `*sql.DB` needs a connection string; there's no sensible zero
value. Forcing one would just move the error later. A constructor that
can fail (`New() (*T, error)`) is correct there.

**Note the asymmetry**: a nil *map* panics on write but reads fine; a nil
*slice* is fully usable for both `append` and `range`. So slices already
have a useful zero value and maps don't — which is why `var s []int` is
idiomatic and `var m map[string]int` usually isn't.

## Links

- Module 04 (zero values, nil map vs nil slice)
- Module 12 (`sync.Mutex`, `sync.Map` zero values)
- Related: `notes/01-effective-go/interfaces-should-be-small.md`
