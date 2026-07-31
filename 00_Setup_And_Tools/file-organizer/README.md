# Project 3 — File Organizer

The most "real" project of the three: it touches the filesystem, which means it
introduces the habit of **planning before acting** — a pattern you'll reuse in any
tool that mutates state (deployments, migrations, batch renames, etc).

## Try It

```bash
mkdir messy-folder && cd messy-folder
touch photo.jpg notes.pdf song.mp3 script.go archive.zip mystery.xyz
cd ..

go run main.go -dir ./messy-folder
```

Output (dry run — nothing is touched yet):
```
Found 6 file(s) in ./messy-folder

  archive.zip  ->  messy-folder/Archives/archive.zip
  mystery.xyz  ->  messy-folder/Other/mystery.xyz
  notes.pdf    ->  messy-folder/Documents/notes.pdf
  photo.jpg    ->  messy-folder/Images/photo.jpg
  script.go    ->  messy-folder/Code/script.go
  song.mp3     ->  messy-folder/Audio/song.mp3

Dry run only — nothing was moved. Re-run with -apply to execute this plan.
```

Happy with the plan? Run it for real:
```bash
go run main.go -dir ./messy-folder -apply
```

## Architecture: Plan, Then Apply

```
┌───────────────────────────────────────────────────────────────┐
│                                                                  │
│   os.ReadDir(dir)                                                │
│         │                                                        │
│         ▼                                                        │
│   buildPlan()  ──▶  []plan{ source, dest }   (pure, no side      │
│         │                                     effects — just     │
│         │                                     figures out what   │
│         │                                     SHOULD happen)     │
│         ▼                                                        │
│   print the plan to the user                                     │
│         │                                                        │
│    -apply flag?                                                  │
│    ┌────┴────┐                                                   │
│    no        yes                                                 │
│    │          │                                                  │
│    ▼          ▼                                                  │
│  stop     apply(plans)  ──▶  os.MkdirAll + os.Rename per file     │
│                               (the ONLY function that touches     │
│                                the real filesystem)               │
└───────────────────────────────────────────────────────────────┘
```

Notice `buildPlan` never calls `os.Rename` or `os.MkdirAll` — it only *reads* the
directory and computes what *should* happen. `apply` is the only function that
mutates anything. This separation is what makes the dry-run flag possible with
almost no extra code, and it's the same reason `terraform plan` / `terraform apply`
exist as two separate steps, or why database migration tools show you the SQL
before running it.

## Idiomatic Go Patterns Used Here

1. **Wrapped errors** — `fmt.Errorf("reading directory: %w", err)` wraps the
   underlying error while adding context, without losing the original error's
   identity (`errors.Is` / `errors.Unwrap` still work on it).
2. **A map as a lookup table** — `categoryFor` is far more maintainable than a
   long `if/else` or `switch` chain, and it's trivial to extend: just add a line.
3. **Struct instead of tuples** — the `plan` struct (rather than two parallel
   slices of strings) keeps `source` and `dest` locked together, so they can
   never accidentally get out of sync.
4. **Table-driven design over hardcoded logic** — this is a very common Go idiom,
   and it's also how most of Go's own test suites are written (a slice of
   input/expected-output pairs, looped over).

## Case Study: Why `os.Rename` and Not "Copy Then Delete"

`os.Rename` is a single atomic filesystem operation *when source and destination
are on the same disk/volume* — the move either fully succeeds or fully fails,
with no in-between state where a file exists in both places or neither. A
"copy, verify, then delete original" approach is what you'd reach for only if
you needed to move data **across** volumes/drives, where an atomic rename isn't
possible at the OS level — and even then, you'd want your own checksum-verify
step, which is a good deal more code than this project needs.

## Try It Yourself
- Add a `-copy` flag that copies instead of moves (use `io.Copy` for this)
- Add recursive scanning of subfolders (`filepath.WalkDir` instead of `os.ReadDir`)
- Add a `-undo` mode that reads the last plan back from a saved log file and
  reverses it
- Handle name collisions: what should happen if `Images/photo.jpg` already exists?
