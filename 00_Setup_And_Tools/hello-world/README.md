# Project 1 — Hello World

The smallest complete Go program, but every line matters. This is the moment to get
comfortable with Go's structure, because every program you'll ever write follows it.

## The Code

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
```

## Line-by-Line Breakdown

```
┌────────────────────────────────────────────────────────────┐
│  package main                                                │
│  └─▶ Declares this file belongs to package "main".           │
│      "main" is special: it tells Go "this produces an        │
│      executable", not a reusable library.                    │
│                                                                │
│  import "fmt"                                                 │
│  └─▶ Pulls in the standard library's formatting package.     │
│      Go refuses to compile if you import something you       │
│      don't use — no dead imports allowed.                    │
│                                                                │
│  func main() { ... }                                          │
│  └─▶ The entry point. Every executable Go program needs      │
│      exactly one func main() inside package main.             │
│      This is where execution starts — like main() in C,      │
│      or the bottom `if __name__ == "__main__":` in Python.    │
└────────────────────────────────────────────────────────────┘
```

## Running It

```bash
cd hello-world
go run main.go
# Hello, World!
```

`go run` compiles to a temp binary, runs it, then throws the binary away — great for
quick iteration. Compare with:

```bash
go build -o hello main.go   # produces a real, standalone binary called "hello"
./hello                      # Hello, World!
```

## Case Study: Why This Matters More Than It Looks

Coming from Python or JavaScript, it might feel odd that Go forces:
1. An explicit package declaration
2. Explicit imports with no wildcard imports
3. A single, unambiguous entry point

This is intentional. Go optimizes for **large teams reading unfamiliar code**. When
you open any `.go` file, you instantly know what package it's in and exactly what it
depends on — no hunting through fifteen files to figure out what's in scope. This
same philosophy shows up everywhere in Go: no unused variables allowed, no unused
imports allowed, `gofmt` enforces one canonical style so there are no style debates
on a team. The language trades a little typing convenience for a lot of long-term
readability — a bet that's paid off at companies running Go at huge scale (Google,
Uber, Cloudflare, Docker's own tooling).

## Try It Yourself
- Change the string and re-run with `go run main.go`
- Try `fmt.Printf("Hello, %s! You are %d.\n", "Gopher", 1)` — Go's `Printf` verbs
  (`%s`, `%d`, `%v`, `%T`) are worth memorizing early, you'll use them constantly.
- Remove the `import "fmt"` line and try to compile — see Go refuse, with a clear
  error. This "fail fast and loud" behavior is a running theme.
