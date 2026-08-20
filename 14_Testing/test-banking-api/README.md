# Project — Test Banking API

```
test-banking-api/
├── go.mod
├── main.go                    package main — runs the API manually (optional)
└── bank/
    ├── bank.go                  Account, Bank, TransactionLogger, ParseAmount
    ├── handler.go                 HTTP handler wrapping Bank
    ├── bank_test.go                unit tests, table tests, mocking
    ├── bank_bench_test.go           benchmarks
    ├── amount_fuzz_test.go           fuzz test
    └── handler_test.go                integration test (httptest)
```

The point of this project is the **test suite**, not the banking logic
itself — every technique from Module 14's guide is used somewhere in here,
against real code.

## Running Everything

```bash
cd test-banking-api

go test ./...                       # every unit, table, and integration test
go test -v ./...                     # same, with per-test names and PASS/FAIL

go test -bench=. ./bank/                # every benchmark
go test -bench=. -benchmem ./bank/       # benchmarks + allocation counts

go test -fuzz=FuzzParseAmount -fuzztime=15s ./bank/   # fuzz for 15 seconds

go test -cover ./...                          # quick coverage %
go test -coverprofile=coverage.out ./...        # detailed profile
go tool cover -html=coverage.out                  # opens an HTML report
go tool cover -func=coverage.out                    # per-function %, no browser
```

Optionally, run the API itself:
```bash
go run main.go
# in another terminal:
curl -X POST http://localhost:8080/deposit -d '{"account":"acc1","amount":25}'
```

## What's Demonstrated Where

| File | Techniques |
|---|---|
| `bank_test.go` | Unit tests (`TestAccount_Deposit`), a table test (`TestAccount_Withdraw`, 5 cases via `t.Run`), and mocking (`FakeLogger`, verifying `Log` is called with the right arguments — and *not* called for rejected transactions) |
| `bank_bench_test.go` | `BenchmarkAccount_Deposit`, with `b.ResetTimer()` excluding setup; `BenchmarkAccount_Withdraw`, resetting state every iteration with `b.StopTimer()`/`b.StartTimer()`; a logger-overhead comparison benchmark |
| `amount_fuzz_test.go` | `FuzzParseAmount`, seeded with known edge cases, checking both "doesn't panic" and "whatever parses successfully round-trips to a sane `String()`" |
| `handler_test.go` | Integration tests — real HTTP requests via `httptest.NewServer`, real JSON encoding/decoding, exercising `Bank` and the HTTP layer *together* |

## Case Study: Why `FakeLogger` Also Tests the *Absence* of a Call

```go
func TestDeposit_RejectedAmounts_DoNotLog(t *testing.T) {
	...
	acc.Deposit(-10) // rejected

	if len(logger.calls) != 0 {
		t.Errorf("expected 0 logged calls for a rejected deposit, got %d", len(logger.calls))
	}
}
```

It's tempting to only test the "happy path" — deposit succeeds, logger gets
called, done. But a logger that fires on *every* call attempt regardless of
success — logging a rejected, no-op deposit as if money actually moved —
would be a real bug with real consequences (an inaccurate transaction
history, wrong totals downstream). A hand-written fake makes this trivial to
verify: check the recorded call *count*, not just its contents, for both the
success and failure paths.

## Case Study: The Logger Benchmark Has a Built-In Pitfall — On Purpose

Look closely at `BenchmarkDepositWithLogger`: `FakeLogger.Log` appends to an
unbounded `[]loggedCall` slice, and the benchmark calls `Deposit` (and
therefore `Log`) up to `b.N` times — which the testing framework will scale
into the *millions* for a fast operation like this one. That slice grows
for the entire benchmark run, meaning part of what this benchmark measures
is actually **`append`'s amortized reallocation cost** (Module 04's slice
internals), not purely `Deposit`'s own performance. This is a genuine,
easy-to-miss benchmarking trap: anything your benchmarked code touches that
accumulates state across iterations skews the measurement, and it's worth
comparing `BenchmarkDepositWithLogger` against `BenchmarkAccount_Deposit`
(same operation, `NoopLogger` instead) with `-benchmem` to see the
difference directly, rather than trusting either number blindly.

## Try It Yourself
- Run `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
  and look for any red (uncovered) lines — `Transfer`'s rollback path
  (`to.Deposit` failing after `from.Withdraw` already succeeded) is a good
  candidate to check: is it actually exercised anywhere in the current test
  suite? If not, that's a real gap coverage just found for you
- Add a `mockgen`-generated mock for `TransactionLogger` (per the guide's
  Libraries section) as an alternative to `FakeLogger`, and compare how much
  test code each approach needs for the same assertions
- Extend `FuzzParseAmount` to also fuzz-test `Bank.Transfer`'s amount
  parameter indirectly, by generating random account ID strings too —
  a good exercise in fuzzing a function with more than one parameter
  (`f.Fuzz` accepts multiple arguments after `*testing.T`)
