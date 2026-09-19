# The fewer methods an interface has, the more types can satisfy it — and the more reusable every function accepting it becomes

**Source**: Effective Go + "The bigger the interface, the weaker the abstraction" (Go Proverbs)
**Why I read this**: I'd written a 7-method `Storage` interface and found
that mocking it in tests was painful. Wanted to know if that was a smell.

## The claim, in my words

An interface's power is inversely proportional to its size. `io.Reader`
has one method and is satisfied by files, network connections, HTTP
bodies, strings, byte buffers, and gzip streams — so any function taking
an `io.Reader` works with all of them for free. A 7-method interface is
satisfied by almost nothing, and mocking it means writing 7 stub methods
to test one code path.

## Concrete example

```go
// WEAK — few types satisfy it, painful to mock, hard to reuse
type Storage interface {
    Get(k string) ([]byte, error)
    Set(k string, v []byte) error
    Delete(k string) error
    List() ([]string, error)
    Close() error
    Stats() Stats
    Compact() error
}

// STRONG — compose small interfaces; accept only what you need
type Getter interface{ Get(k string) ([]byte, error) }
type Setter interface{ Set(k string, v []byte) error }

// A function declares the MINIMUM it needs:
func cacheWarm(g Getter, keys []string) { /* only needs Get */ }
// Testing this requires stubbing exactly ONE method.
```

**What actually happened when I applied it**: splitting my `Storage`
interface let the read-path tests stub one method instead of seven, and
the read-only consumers stopped being able to accidentally call `Delete`.

## The Go-specific corollary

**Define interfaces at the consumer, not the producer.** Because Go's
interface satisfaction is implicit (Module 06), the package that *needs*
`Get` declares a one-method `Getter` locally — the storage package
doesn't need to know that interface exists. This is the opposite of
Java's convention and takes a while to feel natural.

## When this applies / doesn't

Applies to interfaces you define. Doesn't mean *never* have a larger
interface — `http.ResponseWriter` has 3 methods and that's correct for
what it does. The test is: are you forcing implementers to provide
methods that particular callers don't use?

## Links

- Module 06 (implicit satisfaction, composition)
- Module 17 (repository pattern — a case where a slightly larger interface is justified)
- Related: `notes/05-stdlib/sync-once-implementation.md`
