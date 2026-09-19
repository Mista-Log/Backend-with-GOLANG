# Standard Library Deep Dives

Reading stdlib source is the highest-leverage Go reading available:
idiomatic by definition, heavily reviewed, solving problems you have.

## Packages worth reading the source of

| Package | Why |
|---|---|
| `strings.Builder` | Tiny, explains exactly why it beats `+=` (Module 23) |
| `sync.Once` | ~20 lines; the double-checked locking is instructive |
| `net/http` Server | How goroutine-per-request is actually implemented |
| `sort` | `sort.Slice`'s reflection, and why `slices.Sort` replaced it |
| `context` | Smaller than expected; the cancellation tree is elegant |
| `encoding/json` | The reflection-based encoder — dense but illuminating |
| `container/heap` | Interface-based design; a masterclass in small interfaces |

## How to read stdlib source productively

Pick a function you've *used* and trace it down two levels. Don't read a
whole package top to bottom — you'll bounce off.

## Notes in this folder

- `sync-once-implementation.md`

## Related modules

Module 06 (interfaces), Module 23 (performance), Module 12 (sync)
