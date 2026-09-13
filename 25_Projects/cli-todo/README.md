# Beginner — CLI Todo

```bash
cd cli-todo
go run . add "Buy milk"
go run . add "Write the Module 25 README"
go run . list
go run . done 1
go run . rm 2
go run . list
```

## What's Demonstrated

- **Crash-safe persistence** — `saveTasks` writes to a temp file in the
  same directory, then atomically renames it over `todo.json` (Module
  10's safe-write pattern). Kill the process mid-write (hard in practice
  for a file this small, but the mechanism is real) and `todo.json` is
  never left half-written.
- **`out := tasks[:0]`** in `cmdRemove` — reslicing to length zero but
  keeping the original capacity, then appending only the kept items,
  avoids allocating a brand new slice just to remove one element (Module
  04's slice-internals habit).
- **A plain `os.Args` command dispatcher** — no external CLI framework
  needed for something this small; Module 19's `cobra` is the natural
  upgrade once subcommands need their own flags, help text, or nested
  subcommands.

## Try It Yourself
- Add a `todo edit <id> "<new title>"` subcommand
- Swap the storage format for SQLite (Module 17) once you want filtering
  ("show only overdue tasks") beyond a simple linear scan
- Add colorized terminal output for done vs. pending tasks
