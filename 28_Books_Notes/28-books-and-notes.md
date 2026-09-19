# 28. Books & Notes

The final module isn't a topic — it's a **system**. Twenty-eight modules
of material is more than anyone retains passively, and the engineers who
keep growing after a course ends are the ones who keep notes that
compound rather than notes that rot.

---

## Why Most Engineering Notes Fail

```
┌──────────────────────────────────────────────────────────┐
│   The three failure modes:                                             │
│                                                                            │
│   1. TRANSCRIPTION — copying a doc's content verbatim. Produces              │
│        no understanding, and the original was already better written.            │
│                                                                                      │
│   2. HOARDING — saving links "to read later." A bookmark folder is                     │
│        not a note; it's a promise you won't keep.                                         │
│                                                                                               │
│   3. WRITE-ONLY — notes that are never re-read. If nothing ever sends                           │
│        you back to a note, writing it was wasted effort.                                          │
└──────────────────────────────────────────────────────────┘
```

**The fix for all three is the same**: every note answers a question you
actually had, in your own words, with a concrete example you wrote
yourself.

---

## The Note Format

Every note in `notes/` follows this shape — copy
`templates/note-template.md` to start a new one:

```markdown
# <Specific claim, not a topic>

**Source**: <link/book + date read>
**Why I read this**: <the actual question that sent you here>

## The claim, in my words
<2-4 sentences. If you can't compress it, you haven't understood it yet.>

## Concrete example
<Code YOU wrote testing the claim — not copied from the source.>

## When this applies / doesn't
<The boundary. This is the part that makes a note useful 6 months later.>

## Links
<Other notes this connects to.>
```

**Title notes as claims, not topics.** "Channels" is a topic and tells you
nothing later. "An unbuffered channel send is a synchronization point, not
just a queue" is a claim you can scan and immediately re-derive value from.

---

## The Nine Areas

```
┌──────────────────────────────────────────────────────────┐
│   notes/01-effective-go/     → the canonical style document            │
│   notes/02-go-proverbs/       → Rob Pike's compressed wisdom               │
│   notes/03-memory-model/       → the happens-before guarantees                 │
│   notes/04-go-blog/             → release notes, deep dives, design rationale     │
│   notes/05-stdlib/               → reading the standard library's source             │
│   notes/06-performance/           → measured findings, not folklore                      │
│   notes/07-design-patterns/        → patterns as they appear in REAL Go                     │
│   notes/08-fintech-patterns/        → domain architecture (Module 21's territory)              │
│   notes/09-postmortems/              → how real systems actually fail                             │
└──────────────────────────────────────────────────────────┘
```

Each folder has a **starter note** already written — a real example of the
format, on a genuinely useful claim — plus a `README.md` listing what's
worth reading in that area and which questions to bring to it.

---

## The Habit That Makes It Work

```
┌──────────────────────────────────────────────────────────┐
│   READ with a question  →  WRITE the claim in your own words  →           │
│   TEST it with code you wrote  →  LINK it to notes you already have          │
│                                                                                  │
│   The "test it with code" step is what makes Go notes different from             │
│   notes in most fields: you can VERIFY nearly every claim in minutes.               │
│   A note whose example you actually ran is worth ten you only read.                    │
└──────────────────────────────────────────────────────────┘
```

**Re-read trigger**: whenever you hit a bug or design decision in an area
you have notes on, open the note *first*. If it didn't help, the note was
wrong or too vague — fix it right then. That feedback loop is the whole
system; without it you're back to write-only notes.

---

## Notes as Writing Material

If you write publicly about engineering, notes in this format are already
80% of a draft — a claim, an example, and the boundary conditions is
exactly the skeleton of a good technical post. The reverse is also true:
committing to explain something publicly is the most reliable way to
discover you didn't actually understand it.

The `postmortems` folder is especially rich here — real incident reports
from large engineering teams are among the most under-read technical
writing available, and a note connecting one to a pattern you've built
yourself (Module 22's circuit breaker, Module 21's idempotency) is a
genuinely original contribution rather than a summary.

---

## Where to Actually Start

Don't fill all nine folders at once. Pick the area matching whatever
you're building next month, and write **three** notes properly. A small
set of well-tested, re-read notes beats a large set of transcriptions,
every time.
