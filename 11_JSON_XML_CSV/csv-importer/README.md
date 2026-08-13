# Project 2 — CSV Importer

```bash
cd csv-importer
go run main.go
```

Generates `data/products.csv`, imports it into `[]Product`, exports the same
data as both `data/products.json` and `data/products.xml`, then re-parses
each output and confirms every price round-trips exactly.

## What's Demonstrated Here

- **One `ParseMoney` function, three entry points** — CSV import,
  `Money.UnmarshalJSON`, and `Money.UnmarshalXML` all call the same
  `ParseMoney`, instead of three separate (and potentially inconsistent)
  parsing implementations for "turn a string like `$19.99` into cents."
- **`Money` implements custom hooks for BOTH JSON and XML** —
  `MarshalJSON`/`UnmarshalJSON` (the `encoding/json` interfaces) *and*
  `MarshalXML`/`UnmarshalXML` (the analogous, differently-shaped
  `encoding/xml` interfaces) on the same type. Different method signatures,
  same underlying goal: control the wire representation instead of
  accepting the library's default.
- **Symmetric round-tripping, verified in code** — `main` doesn't just
  claim `Marshal`/`Unmarshal` agree, it actually writes the export, reads it
  back, and compares every price to the original — turning the guide's
  symmetry warning into a real, checkable assertion instead of a rule you
  just have to trust.
- **One `Product` struct, three formats** — `json:"..."`, `xml:"..."` tags,
  and manual CSV column mapping all describe the same fields.

```
┌──────────────────────────────────────────────────────────┐
│                       ParseMoney("$19.99")                       │
│                              │                                       │
│              ┌───────────────┼───────────────┐                        │
│              ▼               ▼               ▼                          │
│      CSV import      Money.UnmarshalJSON  Money.UnmarshalXML                │
│      (row parsing)    (JSON import)          (XML import)                      │
│                                                                                     │
│   All three ends up at the SAME Money(1999) value — one parsing                       │
│   implementation, reused everywhere a Money needs to come FROM text.                     │
└──────────────────────────────────────────────────────────┘
```

## Case Study: `encoding/xml`'s Custom Hooks Aren't the Same Shape as JSON's

It's tempting to assume `MarshalXML`/`UnmarshalXML` work identically to
`MarshalJSON`/`UnmarshalJSON` just with a different name — they don't:

```go
// JSON: operates on raw bytes directly
func (m Money) MarshalJSON() ([]byte, error)
func (m *Money) UnmarshalJSON(data []byte) error

// XML: operates through an Encoder/Decoder and an xml.StartElement,
// because XML elements have more structure to manage (start/end tags,
// nesting) than a JSON value does
func (m Money) MarshalXML(e *xml.Encoder, start xml.StartElement) error
func (m *Money) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error
```

`e.EncodeElement(m.String(), start)` and `d.DecodeElement(&s, &start)` are
doing the analogous job to raw byte manipulation in the JSON versions, just
through XML's more structured API — worth having both side by side once, as
this project does, so the *shape* difference is obvious instead of a
surprise the first time you need to write an XML one.

## Try It Yourself
- Add a `Category` field with a completely different XML representation
  than JSON (e.g., an XML *attribute* but a JSON *nested object*) — a good
  exercise in how independently the two tag systems can diverge on the same
  struct
- Deliberately introduce a bug — round `ParseMoney` down instead of to the
  nearest cent — and watch the round-trip check in `main` catch it (or not,
  depending on which values you test) as a demonstration of why that check
  exists at all
- Add a YAML export too, following the guide's `gopkg.in/yaml.v3` example —
  you'll need to run `go get gopkg.in/yaml.v3` first, and note that YAML's
  custom-marshal interface (`yaml.Marshaler`/`yaml.Unmarshaler`) is a third
  distinct shape, different again from both JSON's and XML's
