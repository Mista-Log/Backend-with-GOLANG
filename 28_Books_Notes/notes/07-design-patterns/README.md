# Design Patterns in Go

Go resists several classic OOP patterns (no inheritance, no generics
until 1.18) and makes others nearly invisible. Note the Go-native form,
not the Java textbook form.

## Patterns that show up constantly in real Go

| Pattern | Go form |
|---|---|
| Strategy | An interface + swappable implementations (Module 06) |
| Decorator | Middleware: `func(http.Handler) http.Handler` (Module 15) |
| Factory | A plain `NewX()` constructor function |
| Singleton | `sync.Once` + package-level var (Module 12) |
| Observer | Channels, or a pub/sub broker (Module 20) |
| Builder | Functional options: `WithTimeout(d)` (Module 08's retry) |
| Repository | An interface hiding the data store (Module 17) |
| Composite | A type satisfying the same interface it aggregates (Module 06) |

## Patterns to be suspicious of in Go

- Deep inheritance hierarchies — Go has embedding, not inheritance
- Abstract factories — usually just a function in Go
- Heavy DI frameworks — constructor injection is normally enough (Module 19)

## Notes in this folder

- `functional-options.md`

## Related modules

Module 06 (interfaces), Module 15 (middleware), Module 17 (repository), Module 19 (DI)
