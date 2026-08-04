# Project 2 — Student Management

```bash
cd student-management
go run main.go
```

Add a regular student and a graduate student, give both a few grades, then
list everyone and check the class average.

## What's Demonstrated Here

- **Nested struct (`Address`)** — always accessed through its field name:
  `student.Address.City`. This models "a Student *has an* Address."
- **Embedded struct (`GraduateStudent` embeds `Student`)** — no field name,
  so `gradStudent.Name`, `gradStudent.Grades`, and even `gradStudent.AddGrade(...)`
  all work directly, promoted straight from `Student`. This models
  "a GraduateStudent *is a* Student, plus some extra fields" through
  composition.
- **A promoted METHOD, not just fields** — `addGrade` in the `school` type
  calls `g.AddGrade(grade)` on a `*GraduateStudent` even though `AddGrade` is
  defined on `*Student`. Nothing extra was written for `GraduateStudent` to
  get this — that's the whole point of embedding a struct that already has
  methods.

```
┌──────────────────────────────────────────────────────────┐
│  type Student struct { ID, Name, Address, Grades }            │
│  func (s *Student) AddGrade(g float64) { ... }                    │
│                                                                       │
│  type GraduateStudent struct {                                          │
│      Student          ◀── embedded                                        │
│      ThesisTitle string                                                     │
│      Advisor     string                                                       │
│  }                                                                                │
│                                                                                      │
│  g := GraduateStudent{...}                                                            │
│  g.AddGrade(3.8)   ◀── calls Student's AddGrade, promoted — GraduateStudent            │
│                        never declared this method itself                                 │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Nested vs. Embedded, Side by Side

This project uses **both** in the same file, which makes the difference easy
to see directly:

```go
type Student struct {
	Address Address   // NESTED — has a field name ("Address")
}
s.Address.City         // must go through the field name

type GraduateStudent struct {
	Student            // EMBEDDED — no field name, just the type
}
g.Name                  // promoted — reads as if GraduateStudent declared Name itself
g.Student.Name           // the explicit path still works too, if you ever need it
```

Reach for **nesting** (a named field) when the relationship is genuinely
"this struct contains that struct as one piece of its data" — an `Address`
isn't "a kind of" `Student`, it's just data the student has. Reach for
**embedding** when you want "this struct extends that struct's fields and
behavior" without repeating them — `GraduateStudent` really is a fuller kind
of `Student`. If you're ever unsure which to use: nesting is always safe;
reach for embedding specifically when you want that promotion behavior.

## Try It Yourself
- Add a `TopStudent() (*Student, error)` on `school` that ranges over both
  maps and returns whichever has the highest `Average()`
- Give `GraduateStudent` its *own* `AddGrade` override that also logs "grad
  student X received a grade" — and confirm calling `g.AddGrade(...)` now
  uses the override, not the promoted one (this is the "shadowing" rule from
  the guide's Embedding section)
- Add a `[]string` field for extracurricular activities, and print it in
  `printAll` only when non-empty
