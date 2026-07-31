# Go for Beginners — Module 00: Setup & Environment

## Contents

1. **[00-setup-and-environment.md](./00-setup-and-environment.md)** — Introduction,
   installing Go, GOPATH vs GOROOT, Go Modules, VS Code, GoLand, the CLI, debugging
   with Delve, Go Workspaces, environment variables, build commands, and cross
   compilation. Diagrams included throughout.

2. **[hello-world/](./hello-world)** — The smallest complete Go program, broken down
   line by line, plus a case study on why Go's structure is so strict.

3. **[calculator-cli/](./calculator-cli)** — A CLI with both flag-driven and
   interactive (REPL) modes. Introduces the `flag` package, multiple return values,
   and Go's explicit error-handling model.

4. **[file-organizer/](./file-organizer)** — A real filesystem tool that sorts files
   into folders by type, with a dry-run safety mode. Introduces `os`, `path/filepath`,
   wrapped errors, and the "plan, then apply" architecture pattern.

## Suggested Order

Work through them in the order above — each project is a small step up in
difficulty and introduces exactly one or two new ideas on top of the last.

```
Setup guide  ──▶  Hello World  ──▶  Calculator CLI  ──▶  File Organizer
 (environment)     (structure,        (flags, errors,      (filesystem,
                    entry point)        REPL loop)           safe mutation
                                                              patterns)
```

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go            # run directly, no binary left behind
go build -o app main.go   # compile a standalone binary
./app                     # run the compiled binary
```
