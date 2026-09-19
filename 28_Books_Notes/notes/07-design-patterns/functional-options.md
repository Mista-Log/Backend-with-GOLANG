# Functional options give a constructor optional, named, extensible parameters without breaking existing callers

**Source**: Dave Cheney, "Functional options for friendly APIs" + Module 08's retry package
**Why I read this**: Saw `WithTimeout(...)`-style APIs in gRPC and the AWS
SDK and wanted to understand the pattern properly rather than copy it.

## The claim, in my words

Go has no default or named parameters. Functional options solve this: an
option is a function that mutates a config struct, and the constructor
takes a variadic slice of them. Adding a new option later never breaks
existing callers, because they simply don't pass it.

## Concrete example

```go
type config struct {
    timeout time.Duration
    retries int
}

type Option func(*config)

func WithTimeout(d time.Duration) Option { return func(c *config) { c.timeout = d } }
func WithRetries(n int) Option           { return func(c *config) { c.retries = n } }

func New(opts ...Option) *Client {
    cfg := config{timeout: 30 * time.Second, retries: 3}  // defaults
    for _, opt := range opts { opt(&cfg) }
    return &Client{cfg: cfg}
}

// Callers pass only what they care about, in any order, named at the call site:
New()
New(WithTimeout(5 * time.Second))
New(WithRetries(10), WithTimeout(time.Second))
```

## Why this beats the alternatives

| Alternative | Problem |
|---|---|
| `New(timeout, retries, x, y)` | Adding a param breaks every caller; unreadable at call site |
| `New(cfg Config)` | Callers must construct a struct; zero values are ambiguous |
| Setter methods after New | Object exists in a partially-configured state |

## When this applies / doesn't

**Applies** to constructors with several optional parameters, especially
in a library where you'll add more over time.

**Doesn't apply** to 1-2 required parameters — `New(addr string)` is
clearer than `New(WithAddr(addr))`. Required things should be required
positional params; options are for *optional* things.

## Links

- Module 08 (the retry package uses exactly this pattern)
- Module 03 (higher-order functions — an Option is one)
