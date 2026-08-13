# 11. Serialization

Serialization is converting an in-memory Go value into a portable format
(bytes, text) — and back. Go's standard library covers JSON, CSV, and XML
natively; YAML needs a small, extremely common third-party package. This
module also covers customizing exactly how a type serializes, beyond the
default field-by-field behavior.

---

## 1. JSON

The most common format in modern Go code, handled by `encoding/json`. Struct
fields map to JSON keys via **struct tags**:

```go
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"` // omitted from output if empty
	Password string `json:"-"`                 // NEVER serialized, ever
}
```

```
┌────────────────────────────────────────────────────┐
│   `json:"name"`         →  use "name" as the JSON key,     │
│                              instead of the Go field name         │
│   `json:"email,omitempty"` →  also OMIT this key entirely if         │
│                              the field is its type's zero value         │
│   `json:"-"`             →  NEVER include this field, no matter           │
│                              what — the standard way to keep a               │
│                              field (like Password) out of any JSON             │
│                              output entirely                                       │
└────────────────────────────────────────────────────┘
```

Only **exported** fields (capitalized — Module 03) are ever serialized;
`encoding/json` can't see unexported fields at all, tags or not.

---

## 2. CSV

Covered hands-on in Module 10 — `encoding/csv` reads/writes rows as
`[]string`, with no built-in struct mapping (unlike JSON/XML). You map
columns to struct fields yourself, by index or by a header-name lookup (the
`indexColumns` pattern from Module 10's CSV Reader project).

```go
reader := csv.NewReader(f)
records, err := reader.ReadAll() // [][]string — every row is []string
```

---

## 3. XML

Handled by `encoding/xml`, with its own struct tag dialect — noticeably more
expressive than JSON's, since XML distinguishes attributes from nested
elements:

```go
type Book struct {
	XMLName xml.Name `xml:"book"`
	ISBN    string   `xml:"isbn,attr"`     // an ATTRIBUTE: <book isbn="...">
	Title   string   `xml:"title"`          // a nested ELEMENT: <title>...</title>
	Author  string   `xml:"author>name"`     // nested inside another element
}
```

```go
b := Book{ISBN: "978-0-13-468599-1", Title: "The Go Programming Language", Author: "Donovan"}
data, _ := xml.MarshalIndent(b, "", "  ")
// <book isbn="978-0-13-468599-1">
//   <title>The Go Programming Language</title>
//   <author><name>Donovan</name></author>
// </book>
```

```
┌────────────────────────────────────────────────────┐
│   `xml:"isbn,attr"`     →  an XML ATTRIBUTE                │
│   `xml:"title"`          →  a plain nested ELEMENT              │
│   `xml:"author>name"`     →  a nested element INSIDE another        │
│                              element (author > name)                    │
└────────────────────────────────────────────────────┘
```

---

## 4. YAML

**Not** in the standard library — the near-universal choice is the
third-party `gopkg.in/yaml.v3` package (`go get gopkg.in/yaml.v3`). The API
deliberately mirrors `encoding/json`'s shape, just with `yaml` tags:

```go
import "gopkg.in/yaml.v3"

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

data := []byte("host: localhost\nport: 8080\n")
var cfg Config
yaml.Unmarshal(data, &cfg)
```

YAML shows up constantly in Go tooling configuration (Kubernetes manifests,
Docker Compose, CI pipelines) precisely because it's more human-writable
than JSON (comments, no trailing-comma footguns, less punctuation) — but
it's a considerably more complex spec under the hood, so stick to
`encoding/json`/`encoding/xml` for data your *program* produces and
consumes, reaching for YAML mainly where a human is expected to hand-edit
the file.

---

## 5. Encoding

"Encoding" is the Go → format direction: `Marshal` (JSON/XML) turns a Go
value into bytes.

```go
u := User{ID: 1, Name: "Ada", Email: "ada@example.com"}
data, err := json.Marshal(u)          // compact: {"id":1,"name":"Ada","email":"ada@example.com"}
data, err = json.MarshalIndent(u, "", "  ") // pretty-printed, for humans/logs
```

For **streaming** encoding (writing directly to a file, an HTTP response, or
any `io.Writer`, instead of building the whole byte slice in memory first —
recall Module 10's streams section), use an `Encoder`:

```go
enc := json.NewEncoder(os.Stdout)
enc.Encode(u) // writes JSON directly, followed by a newline
```

---

## 6. Decoding

The reverse: bytes → Go value, via `Unmarshal`, or a streaming `Decoder`.

```go
data := []byte(`{"id":1,"name":"Ada","email":"ada@example.com"}`)
var u User
err := json.Unmarshal(data, &u) // note the & — Unmarshal writes THROUGH a pointer

dec := json.NewDecoder(resp.Body) // streaming — good for HTTP response bodies
err = dec.Decode(&u)
```

```
┌────────────────────────────────────────────────────┐
│   Marshal(v)      →  Go value  ──▶  []byte                  │
│   Unmarshal(data, &v) →  []byte  ──▶  Go value (via pointer)      │
│                                                                        │
│   Encoder.Encode(v)        →  Go value  ──▶  io.Writer, STREAMING        │
│   Decoder.Decode(&v)        →  io.Reader  ──▶  Go value, STREAMING          │
│                                                                                    │
│   Prefer Marshal/Unmarshal for small, one-shot data already in memory.               │
│   Prefer Encoder/Decoder when reading/writing directly against a file or                 │
│   network connection — same trade-off as Module 10's streams vs. ReadFile.                   │
└────────────────────────────────────────────────────┘
```

**Unknown fields are silently ignored by default** — decoding JSON with
extra keys your struct doesn't declare doesn't error. Use
`dec.DisallowUnknownFields()` on a `Decoder` if you specifically want
strict decoding that rejects unexpected keys.

---

## 7. Custom Marshal

Implement `MarshalJSON() ([]byte, error)` on a type to fully control how it
serializes — useful when the wire format shouldn't be a literal field-by-field
dump of your Go struct.

```go
type Money int64 // stored as CENTS internally, to avoid float rounding issues

func (m Money) MarshalJSON() ([]byte, error) {
	dollars := float64(m) / 100
	return []byte(fmt.Sprintf(`"$%.2f"`, dollars)), nil // serializes as a formatted STRING
}

type Order struct {
	Total Money `json:"total"`
}

data, _ := json.Marshal(Order{Total: 4999})
// {"total":"$49.99"}   — NOT {"total":4999}
```

```
┌────────────────────────────────────────────────────────┐
│   Without custom MarshalJSON:  Money(4999) → 4999   (raw int)     │
│   WITH custom MarshalJSON:      Money(4999) → "$49.99" (formatted)    │
│                                                                             │
│   json.Marshal automatically calls MarshalJSON() on ANY value that            │
│   has it — this is interface satisfaction again (Module 06): Money             │
│   satisfies `interface { MarshalJSON() ([]byte, error) }` implicitly,             │
│   and encoding/json checks for that interface before falling back to                 │
│   its default field-by-field behavior.                                                    │
└────────────────────────────────────────────────────────┘
```

---

## 8. Custom Unmarshal

The reverse: `UnmarshalJSON([]byte) error` lets a type control exactly how
it parses itself back out of raw JSON — often needed for the same
type that has a custom `MarshalJSON`, so encoding and decoding stay
symmetric.

```go
func (m *Money) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Money must be a JSON string: %w", err)
	}
	s = strings.TrimPrefix(s, "$")
	dollars, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid Money value %q: %w", s, err)
	}
	*m = Money(dollars * 100) // note the *m — writing THROUGH the pointer receiver
	return nil
}

var o Order
json.Unmarshal([]byte(`{"total":"$49.99"}`), &o)
fmt.Println(o.Total) // 4999 (cents)
```

A very common, more general use case for custom unmarshaling: **flexible
input formats** — accepting a timestamp as either a Unix integer or an ISO
string, or a field that might arrive as either a single value or an array,
depending on which upstream system produced the JSON. `UnmarshalJSON` is
where you'd branch on the raw bytes' shape (often via a preliminary
`json.Unmarshal` into `any` or `json.RawMessage`) and normalize it into one
consistent internal representation.

```
┌────────────────────────────────────────────────────────┐
│   Custom MarshalJSON / UnmarshalJSON MUST stay symmetric:              │
│                                                                              │
│   Marshal(Money(4999))          →  "$49.99"                                    │
│   Unmarshal("$49.99")  into Money  →  Money(4999)                                  │
│                                                                                          │
│   If they DISAGREE on the wire format, round-tripping a value through            │
│   Marshal then Unmarshal silently corrupts it — always test both               │
│   directions together, not just one.                                                   │
└────────────────────────────────────────────────────────┘
```

---

Onto the projects — API Client leans on JSON decoding (including a custom
`UnmarshalJSON` for a flexible field) against a local test server, so it
runs deterministically with no real network dependency; CSV Importer takes
one CSV source and serializes it out as both JSON and XML, using a custom
`Money` type to show `MarshalJSON`/`UnmarshalJSON` symmetry directly.
