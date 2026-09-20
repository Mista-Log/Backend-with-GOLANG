# Go for Beginners — Module 28: Books & Notes

## Contents

1. **[28-books-and-notes.md](./28-books-and-notes.md)** — Why most
   engineering notes fail (transcription, hoarding, write-only), the note
   format that avoids all three, and the read → write → test → link habit
   that makes a notes system compound instead of rot.

2. **[templates/](./templates)** — Two templates: `note-template.md` for
   concept notes, `postmortem-template.md` for incident reports (which
   have a genuinely different shape — failure *chains*, not root causes).

3. **[notes/](./notes)** — Nine folders, one per area from the outline.
   Each has a **README** listing what's actually worth reading there and
   which questions to bring to it, plus a **fully-written starter note**
   demonstrating the format on a real, useful claim:

| Folder | Starter note |
|---|---|
| `01-effective-go/` | Interfaces should be small — and defined at the consumer |
| `02-go-proverbs/` | Make the zero value useful |
| `03-memory-model/` | The complete list of happens-before edges |
| `04-go-blog/` | Errors are values — the errWriter pattern |
| `05-stdlib/` | Why `sync.Once` needs *both* an atomic and a mutex |
| `06-performance/` | `strings.Builder` vs `+=`, with real benchmark shape |
| `07-design-patterns/` | Functional options, and when *not* to use them |
| `08-fintech-patterns/` | Money as integers — the float failure, demonstrated |
| `09-postmortems/` | Retry storms need jitter, not just backoff |

## How to Use This

The starter notes are examples of the *format*, not a reading list to
consume. The point is the habit: read with a question, write the claim in
your own words, test it with code you wrote, link it to what you already
know.

**Don't fill all nine folders at once.** Pick the area matching whatever
you're building next month and write three notes properly.

## The Note Format, in Brief

```
# <A specific CLAIM, not a topic name>
**Source** + **Why I read this** (the actual question)
## The claim, in my words          (2-4 sentences, compressed)
## Concrete example                 (code YOU wrote, and what it did)
## When this applies / doesn't       (the boundary — the part people skip)
## Links                              (to other notes, and to modules)
```

Titling notes as **claims** rather than topics is the single highest-value
habit here. "Channels" tells you nothing when scanning six months later;
"An unbuffered channel send is a synchronization point, not a queue" lets
you re-derive the whole idea instantly.

---

## This Is the End of the Course

Twenty-nine modules, 00 through 28 — from `go version` through
fundamentals, concurrency, HTTP, databases, authentication,
microservices, distributed systems, a complete fintech backend,
performance, cloud deployment, system design, interview preparation, and
now a system for keeping everything you learn after this.

The modules are finished. The notes folder isn't meant to be.


## This Is the End of the Course

Twenty-nine modules, 00 through 28 — from `go version` through
fundamentals, concurrency, HTTP, databases, authentication,
microservices, distributed systems, a complete fintech backend,
performance, cloud deployment, system design, interview preparation, and
now a system for keeping everything you learn after this.

The modules are finished. The notes folder isn't meant to be.