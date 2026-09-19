# Effective Go

**Source**: https://go.dev/doc/effective_go

The canonical Go style document. Not a tutorial — a statement of what
idiomatic Go looks like and why. Worth re-reading annually; you notice
different things once you've written more Go.

## Questions worth bringing to it

- Why `MixedCaps` rather than underscores, and how does that interact
  with the exported/unexported rule (Module 03)?
- When is a named return value genuinely clearer, and when is it noise?
- What's the actual guidance on pointer vs. value receivers?
- Why does the doc push so hard on small interfaces?

## Notes in this folder

- `interfaces-should-be-small.md`

## Related modules

Module 03 (exported names), Module 06 (interfaces), Module 19 (package design)
