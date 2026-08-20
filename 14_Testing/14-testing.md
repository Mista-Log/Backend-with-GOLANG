# 14. Testing

Go treats testing as a first-class part of the toolchain, not a bolted-on
framework — `go test` and the `testing` package are built in, with no
external dependency required to write real, useful tests. This module
covers the built-in tooling in depth, plus the two most common external
libraries you'll see in real Go codebases.

---

## 1. Unit Testing

A Go test is an ordinary function named `TestXxx(t *testing.T)`, living in a
file named `xxx_test.go` **in the same package** as the code it tests.

```go
// math.go
package mathutil

func Add(a, b int) int {
	return a + b
}
```

```go
// math_test.go
package mathutil

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result) // reports failure, TEST CONTINUES
	}
}
```

```
┌────────────────────────────────────────────────────┐
│   go test ./...                                            │
│        │                                                       │
│        ▼                                                          │
│   finds every *_test.go file, runs every TestXxx function            │
│                                                                            │
│   t.Errorf(...)  →  marks the test FAILED, keeps running the rest         │
│                       of THIS test function                                   │
│   t.Fatalf(...)   →  marks the test FAILED, stops THIS test                     │
│                       function immediately (use when continuing               │
│                       would panic or make no sense — e.g. after a                │
│                       failed setup step)                                            │
└────────────────────────────────────────────────────┘
```

```bash
go test ./...         # run every test in every package
go test -v ./...       # verbose — prints each test's name and PASS/FAIL
go test -run TestAdd     # run only tests whose name matches this pattern
```

---

## 2. Table Tests

The dominant Go idiom for testing many input/output pairs against the same
logic: a slice of test cases, looped over, each run as its own **subtest**
via `t.Run`.

```go
func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"two positives", 2, 3, 5},
		{"negative and positive", -1, 1, 0},
		{"two negatives", -2, -3, -5},
		{"zeros", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
```

```
┌────────────────────────────────────────────────────┐
│   go test -v -run TestAdd                                  │
│        │                                                       │
│        ▼                                                          │
│   === RUN   TestAdd/two_positives            ← each case runs        │
│   --- PASS: TestAdd/two_positives                 as its OWN named,       │
│   === RUN   TestAdd/negative_and_positive           independently             │
│   --- PASS: TestAdd/negative_and_positive            reportable subtest           │
│   === RUN   TestAdd/two_negatives                                                     │
│   --- PASS: TestAdd/two_negatives                                                        │
└────────────────────────────────────────────────────┘
```

This is exactly the shape you've been writing test cases into throughout
this course (Module 03's mathlib examples, Module 08's validators) — table
tests just formalize "here are several inputs and their expected outputs"
into something `go test` can run, report, and filter individually. One new
failing case shows up as one named failure, not a wall of undifferentiated
output.

---

## 3. Benchmarking

A benchmark measures **performance**, not correctness — named `BenchmarkXxx(b *testing.B)`,
run with `go test -bench`.

```go
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ { // b.N is chosen AUTOMATICALLY by the testing
		Add(2, 3)                // framework — it runs the loop enough times
	}                              // to get a stable, statistically meaningful measurement
}
```

```bash
go test -bench=.              # run every benchmark in the package
go test -bench=Add -benchmem    # also report memory allocations per operation
```

```
BenchmarkAdd-8       1000000000       0.25 ns/op       0 B/op       0 allocs/op
            │              │              │                │            │
        GOMAXPROCS    iterations      time per         bytes         allocations
        during the       actually     iteration        allocated      per iteration
        run             executed                        per iteration
```

Use `b.ResetTimer()` after any expensive setup that shouldn't count toward
the measurement, and `b.ReportAllocs()` (or the `-benchmem` flag) to track
memory — allocation count is often the more actionable number, since it
points directly at where the garbage collector is doing avoidable work.

---

## 4. Fuzz Testing

Fuzzing (Go 1.18+, native in `go test`) automatically generates many
randomized inputs, hunting for ones that crash your code or fail an
assertion — genuinely different from a table test's fixed, human-chosen
cases.

```go
func FuzzParseAmount(f *testing.F) {
	// Seed corpus — a few known interesting starting inputs:
	f.Add("19.99")
	f.Add("0")
	f.Add("-5.00")

	f.Fuzz(func(t *testing.T, input string) {
		result, err := ParseAmount(input) // the function under test
		if err != nil {
			return // an error is a valid outcome for garbage input — not a failure
		}
		// whatever DID parse should round-trip back to a consistent string:
		if result.String() == "" {
			t.Errorf("parsed %q but got an empty String()", input)
		}
	})
}
```

```bash
go test -fuzz=FuzzParseAmount -fuzztime=30s   # fuzz for 30 seconds
go test ./...                                    # regular `go test` ALSO runs
                                                    # every crash the fuzzer found,
                                                    # saved under testdata/fuzz/, as
                                                    # permanent regression cases
```

```
┌────────────────────────────────────────────────────────┐
│   f.Add(...)   →  seed inputs, a starting point                    │
│        │                                                              │
│        ▼                                                                 │
│   go test -fuzz  →  mutates seeds RANDOMLY (bit flips, byte swaps,          │
│                       length changes...), running your function on          │
│                       thousands of generated variants per second               │
│        │                                                                          │
│        ▼                                                                             │
│   any input that CRASHES or FAILS an assertion gets saved to                          │
│   testdata/fuzz/ automatically — future `go test` runs replay it                         │
│   forever as a normal regression test, even without -fuzz                                   │
└────────────────────────────────────────────────────────┘
```

Fuzz testing shines specifically for **parsers, deserializers, and anything
processing untrusted input** — exactly the kind of code where a human
writing table-test cases tends to only think of "reasonable" malformed
input, missing the genuinely weird edge case a fuzzer finds in seconds.

---

## 5. Integration Tests

Unit tests isolate one function/type; integration tests verify **several
real pieces working together** — a full HTTP handler chain, a database
round-trip, a whole package's public API exercised end-to-end. In Go, these
are still just `TestXxx` functions — no separate framework needed — often
using `httptest` (seen throughout Modules 11-13) to stand up a real,
in-process server.

```go
func TestBankAPI_Deposit(t *testing.T) {
	server := httptest.NewServer(NewBankHandler())
	defer server.Close()

	resp, err := http.Post(server.URL+"/deposit", "application/json",
		strings.NewReader(`{"account": "acc1", "amount": 100}`))
	if err != nil {
		t.Fatalf("request failed: %v", err) // Fatalf — no point checking status below
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d; want 200", resp.StatusCode)
	}
}
```

**Separate slow/external integration tests from fast unit tests** using a
build tag, so `go test ./...` stays fast for everyday development, with
integration tests run explicitly (e.g., in CI, or before a release):
```go
//go:build integration

package bank_test
```
```bash
go test ./...                     # skips files tagged `integration`
go test -tags=integration ./...     # includes them
```

---

## 6. Mocking

A **mock** (or simpler **stub**/**fake**) stands in for a real dependency —
a database, an external API, a payment processor — so a test can run fast,
deterministically, and offline, without touching the real thing. This is
where Module 06's interfaces pay off directly: **code written against an
interface can be tested against a fake implementation of that interface**,
with zero changes to the code under test.

```go
type TransactionLogger interface {
	Log(accountID string, amount float64)
}

// The REAL implementation, used in production:
type FileLogger struct{ /* ... */ }
func (f *FileLogger) Log(accountID string, amount float64) { /* writes to a file */ }

// A hand-written FAKE, used only in tests:
type FakeLogger struct {
	calls []string
}
func (f *FakeLogger) Log(accountID string, amount float64) {
	f.calls = append(f.calls, fmt.Sprintf("%s:%.2f", accountID, amount))
}
```

```go
func TestWithdraw_LogsTransaction(t *testing.T) {
	logger := &FakeLogger{}
	account := NewAccount("acc1", 100, logger)

	account.Withdraw(30)

	if len(logger.calls) != 1 {
		t.Fatalf("expected 1 logged call, got %d", len(logger.calls))
	}
	if logger.calls[0] != "acc1:30.00" {
		t.Errorf("logged call = %q; want \"acc1:30.00\"", logger.calls[0])
	}
}
```

```
┌────────────────────────────────────────────────────────┐
│   Production:  Account  ──▶  TransactionLogger  ──▶  FileLogger        │
│                                                                             │
│   Test:        Account  ──▶  TransactionLogger  ──▶  FakeLogger             │
│                                                                                   │
│   Account's code is IDENTICAL in both — it only ever calls the                      │
│   TransactionLogger INTERFACE, never knows or cares which concrete                     │
│   implementation it's actually talking to (Module 06's implicit                           │
│   satisfaction, put to direct practical use)                                                 │
└────────────────────────────────────────────────────────┘
```

A hand-written fake like `FakeLogger` above is often all you need — it's
simple, has zero dependencies, and is easy to read. For interfaces with many
methods, or when you want to assert on call *order*, specific *arguments*
per call, or generate the boilerplate automatically, reach for `gomock` (see
Libraries below).

---

## 7. Coverage

`go test`'s built-in coverage tooling reports exactly which lines your
tests actually exercised:

```bash
go test -cover ./...                          # quick summary: coverage % per package
go test -coverprofile=coverage.out ./...        # detailed, line-by-line profile
go tool cover -html=coverage.out                  # opens an HTML report in your
                                                     # browser — GREEN lines were
                                                     # covered, RED lines weren't
go tool cover -func=coverage.out                    # per-function coverage as text,
                                                       # no browser needed
```

```
┌────────────────────────────────────────────────────┐
│   ok  	mathutil	0.003s	coverage: 87.5% of statements    │
│                                                                    │
│   87.5% means 87.5% of executable STATEMENTS ran at least              │
│   once during the test suite — it says NOTHING about whether            │
│   the ASSERTIONS in those tests are actually meaningful,                   │
│   or whether every important EDGE CASE was covered                           │
└────────────────────────────────────────────────────┘
```

**High coverage is not the same as a good test suite.** 100% coverage with
weak assertions (or none at all) proves your code *ran*, not that it's
*correct* — a test calling `Withdraw(30)` and checking nothing afterward
"covers" the function while verifying nothing useful. Treat coverage as a
tool for finding **untested code paths you forgot about**, not as a target
to maximize for its own sake.

---

## 8. Libraries

### `testing`

Everything above uses only the standard library's `testing` package — no
import beyond `"testing"` needed for unit tests, table tests, benchmarks,
fuzz tests, or `httptest`-based integration tests. This is genuinely
sufficient for the large majority of real Go test suites.

### `testify`

The most widely-used third-party testing library — mainly for more
expressive **assertions** and lightweight mocking:

```bash
go get github.com/stretchr/testify
```

```go
import "github.com/stretchr/testify/assert"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	assert.Equal(t, 5, result, "Add(2, 3) should equal 5") // fails but CONTINUES the test
	// require.Equal(...) is the same idea, but fails AND stops the test immediately,
	// analogous to t.Fatalf vs t.Errorf
}
```

Compare this against the raw `if result != 5 { t.Errorf(...) }` from Section
1 — `testify/assert` trades a little indirection for noticeably terser,
more readable assertions once a test file has many checks in it.

### `gomock`

A code-generation-based mocking framework — you define an interface,
`mockgen` generates a full fake implementation with call recording,
argument matching, and call-count expectations built in:

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=logger.go -destination=mock_logger.go -package=bank
```

```go
func TestWithdraw_LogsTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := NewMockTransactionLogger(ctrl) // generated by mockgen

	mockLogger.EXPECT().Log("acc1", 30.0).Times(1) // asserts THIS exact call happens ONCE

	account := NewAccount("acc1", 100, mockLogger)
	account.Withdraw(30)
	// ctrl verifies the expectation automatically when the test function ends
}
```

```
┌────────────────────────────────────────────────────────┐
│   Hand-written fake (Section 6):   simple, zero deps, you WRITE            │
│                                      the call-recording logic yourself          │
│                                                                                       │
│   gomock:                           generated, more powerful (argument                 │
│                                      matchers, call ORDER, call COUNT                       │
│                                      expectations) — but needs the                             │
│                                      mockgen tool and a code-gen step                              │
│                                      in your workflow                                                 │
└────────────────────────────────────────────────────────┘
```

**Default to a hand-written fake for a small number of simple interfaces**
(as this module's project does); reach for `gomock` once you have many
interfaces to mock, or need precise assertions about call order/count/exact
arguments that a hand-written fake would take real effort to replicate
correctly.

---

Onto the project — Test Banking API is a real package with a full test
suite covering every technique above: unit tests, table tests, a benchmark,
a fuzz test, a hand-written mock verifying logging behavior, an
`httptest`-based integration test, and instructions for checking its own
coverage.
