# Performance Notes

The rule for this folder: **every note records a measurement**, not a
belief. Go performance folklore is unusually widespread and unusually
often wrong for your specific workload.

## Note requirements for this folder

- The benchmark code, verbatim
- The actual `-benchmem` output, pasted
- Your machine/Go version (numbers aren't portable)
- What you *expected* before running it — being wrong is the useful part

## Sources worth reading

- Dave Cheney's "High Performance Go" workshop (free online)
- The `pprof` docs + `go tool pprof` help
- Go release notes — compiler/GC improvements land regularly and can
  invalidate an old note
- "Profiling Go Programs" on the Go blog

## Notes in this folder

- `strings-builder-vs-concat.md`

## Related modules

Module 23 (profiling, escape analysis), Module 14 (benchmarking), Module 04 (slices)
