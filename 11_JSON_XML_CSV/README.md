# Go for Beginners — Module 11: Serialization

## Contents

1. **[11-serialization.md](./11-serialization.md)** — JSON (`encoding/json`
   struct tags: naming, `omitempty`, `-`), CSV (no built-in struct mapping —
   you write the conversion), XML (`encoding/xml`'s richer tag dialect:
   elements, attributes, nested paths), YAML (the standard third-party
   `gopkg.in/yaml.v3` package), encoding (`Marshal`/`Encoder`), decoding
   (`Unmarshal`/`Decoder`), and custom `MarshalJSON`/`UnmarshalJSON` — with a
   warning about keeping them symmetric. Diagrams included throughout.

2. **[api-client/](./api-client)** — A local `httptest` server (fully
   offline, deterministic) serving the same data as JSON and XML. Covers
   streaming decode from an HTTP response body, encoding a JSON request
   body, and a custom `UnmarshalJSON` (`FlexibleTime`) that accepts several
   upstream timestamp formats — with a case study on why it's *deliberately*
   asymmetric with `Marshal` (Postel's Law), unlike the next project's
   `Money` type.

3. **[csv-importer/](./csv-importer)** — Imports products from CSV, exports
   the same data as JSON *and* XML through a `Money` type with custom hooks
   for **both** formats (`MarshalJSON`/`UnmarshalJSON` *and*
   `MarshalXML`/`UnmarshalXML` — different interface shapes, same goal), all
   funneling through one shared `ParseMoney` function. `main` actually
   performs a round-trip check (export → re-import → compare) instead of
   just asserting symmetry holds.

## Suggested Order

```
Serialization guide ──▶ API Client ──▶ CSV Importer
                          (JSON + XML in,      (CSV in, JSON + XML out,
                           custom Unmarshal      symmetric custom Marshal
                           only, on purpose)      AND Unmarshal, verified)
```

The two projects are a deliberate pair: API Client shows when asymmetric
custom marshaling is the *right* choice (absorbing messy external input);
CSV Importer shows the more common case, where a custom type's wire format
should round-trip exactly, and proves it in code rather than just asserting it.

## Quick Reference: Running Either Project

```bash
cd api-client && go run main.go
cd csv-importer && go run main.go
```

Both are fully self-contained — API Client's server is local and in-process;
CSV Importer generates its own sample input. Nothing to set up beforehand.

*Note: this module builds on Modules 00–10 — start there first if you
haven't already, especially Module 06 (interfaces — custom Marshal/Unmarshal
are implicit interface satisfaction) and Module 10 (streaming reads, which
this module extends to HTTP response bodies).*
