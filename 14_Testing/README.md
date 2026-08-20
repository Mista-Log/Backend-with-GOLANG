# Go for Beginners — Module 14: Testing

## Contents

1. **[14-testing.md](./14-testing.md)** — Unit testing (`TestXxx`,
   `t.Errorf` vs. `t.Fatalf`), table tests (`t.Run` subtests), benchmarking
   (`b.N`, `-benchmem`), fuzz testing (Go 1.18+ native fuzzing, seed corpus,
   automatic regression cases under `testdata/fuzz/`), integration tests
   (`httptest`, build tags to separate them from fast unit tests), mocking
   (hand-written fakes against an interface — the direct payoff of Module
   06), coverage (`-cover`, `-coverprofile`, the HTML report, and a warning
   about what coverage percentage does and doesn't prove), and the two most
   common libraries beyond the standard `testing` package: `testify` and
   `gomock`. Diagrams throughout.

2. **[test-banking-api/](./test-banking-api)** — A real `bank` package (
   `Account`, `Bank`, an HTTP handler) with a full test suite covering every
   technique from the guide against actual code: unit tests, a 5-case table
   test, a hand-written mock (`FakeLogger`) verifying both that logging
   *happens* correctly and that it *doesn't* happen for rejected
   transactions, four benchmarks (including one with a deliberate,
   discussed measurement pitfall), a fuzz test for a currency parser, and
   `httptest`-based integration tests exercising the real HTTP layer.

## Suggested Order

```
Testing guide ──▶ Test Banking API
```

This module has one project because the project itself *is* the survey —
every technique in the guide gets a corresponding, runnable example against
the same small codebase, rather than being split across several smaller
projects. Read the guide section, then find its counterpart in
`bank_test.go` / `bank_bench_test.go` / `amount_fuzz_test.go` /
`handler_test.go`.

## Quick Reference: Running the Test Suite

```bash
cd test-banking-api

go test ./...                                          # unit + table + integration tests
go test -bench=. -benchmem ./bank/                        # benchmarks
go test -fuzz=FuzzParseAmount -fuzztime=15s ./bank/          # fuzz for 15s
go test -coverprofile=coverage.out ./... && \
  go tool cover -html=coverage.out                             # coverage, as an HTML report
```

*Note: this module builds on Modules 00–13 — start there first if you
haven't already, especially Module 06 (interfaces — mocking is implicit
interface satisfaction, applied to tests) and Module 11 (`httptest`, used
throughout this module's integration tests). `testify` and `gomock`
(mentioned in the guide's Libraries section) are external packages — run
`go get github.com/stretchr/testify` or
`go install go.uber.org/mock/mockgen@latest` yourself if you want to try
the "Try It Yourself" suggestions that use them; the project itself only
needs the standard library.*
