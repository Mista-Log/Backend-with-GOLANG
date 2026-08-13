// Project 2: CSV Importer
//
// Imports products from CSV into typed Go structs, then exports the SAME
// data as both JSON and XML — the most common real-world serialization
// pipeline: ingest one format, work with typed structs, emit others. Money
// gets custom Marshal/Unmarshal for BOTH JSON and XML, kept deliberately
// symmetric (unlike API Client's FlexibleTime), with a round-trip check in
// main to prove it.
package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Money stores cents internally — see Module 11's guide for why (avoiding
// float rounding errors on currency) — but serializes as a normal decimal
// string on both the JSON and XML sides.
type Money int64

// ParseMoney is shared logic: CSV parsing, JSON UnmarshalJSON, and XML
// UnmarshalXML all funnel through this ONE function, instead of three
// separate, potentially-inconsistent parsing implementations.
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "$"))
	dollars, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid money value %q: %w", s, err)
	}
	return Money(dollars*100 + 0.5), nil // +0.5 for simple round-to-nearest-cent
}

func (m Money) String() string {
	return fmt.Sprintf("$%.2f", float64(m)/100)
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, m.String())), nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Money must be a JSON string: %w", err)
	}
	parsed, err := ParseMoney(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// MarshalXML/UnmarshalXML are XML's equivalent hooks — a different
// interface (xml.Marshaler/xml.Unmarshaler) than JSON's, but the same
// underlying idea: control exactly how this type reads and writes itself.
func (m Money) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(m.String(), start)
}

func (m *Money) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	parsed, err := ParseMoney(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// Product is tagged for JSON, XML, AND used with manual CSV mapping —
// three serialization formats, one struct.
type Product struct {
	XMLName  xml.Name `json:"-" xml:"product"`
	SKU      string   `json:"sku" xml:"sku,attr"`
	Name     string   `json:"name" xml:"name"`
	Price    Money    `json:"price" xml:"price"`
	Quantity int      `json:"quantity" xml:"quantity"`
}

type Catalog struct {
	XMLName  xml.Name  `json:"-" xml:"catalog"`
	Products []Product `json:"products" xml:"product"`
}

func generateSampleCSV(path string) error {
	content := `sku,name,price,quantity
SKU-001,Wireless Mouse,$19.99,150
SKU-002,Mechanical Keyboard,$89.50,75
SKU-003,USB-C Hub,$34.00,200
SKU-004,Webcam 1080p,$45.99,60
`
	return os.WriteFile(path, []byte(content), 0644)
}

// importProductsCSV streams the CSV (Module 10's habit) and converts each
// row into a Product, routing the price column through the SAME ParseMoney
// function the JSON/XML custom unmarshalers use — one parsing
// implementation, three entry points.
func importProductsCSV(path string) ([]Product, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	col := make(map[string]int, len(header))
	for i, name := range header {
		col[name] = i
	}

	var products []Product
	for {
		row, err := reader.Read()
		if err != nil {
			break // includes io.EOF — Module 10/11 covered checking this properly;
			         // kept simple here since the focus is the Money round-trip
		}

		price, err := ParseMoney(row[col["price"]])
		if err != nil {
			return products, fmt.Errorf("row %v: %w", row, err)
		}
		qty, err := strconv.Atoi(row[col["quantity"]])
		if err != nil {
			return products, fmt.Errorf("row %v: invalid quantity: %w", row, err)
		}

		products = append(products, Product{
			SKU:      row[col["sku"]],
			Name:     row[col["name"]],
			Price:    price,
			Quantity: qty,
		})
	}
	return products, nil
}

func exportJSON(catalog Catalog, path string) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func exportXML(catalog Catalog, path string) error {
	data, err := xml.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func main() {
	const inputCSV = "data/products.csv"
	const outputJSON = "data/products.json"
	const outputXML = "data/products.xml"

	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Println("Error:", err)
		return
	}
	if err := generateSampleCSV(inputCSV); err != nil {
		fmt.Println("Error generating sample CSV:", err)
		return
	}

	fmt.Println("=== Importing from CSV ===")
	products, err := importProductsCSV(inputCSV)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, p := range products {
		fmt.Printf("  %-8s %-22s %-10s qty:%d\n", p.SKU, p.Name, p.Price, p.Quantity)
	}

	catalog := Catalog{Products: products}

	fmt.Println("\n=== Exporting as JSON and XML ===")
	if err := exportJSON(catalog, outputJSON); err != nil {
		fmt.Println("Error:", err)
		return
	}
	if err := exportXML(catalog, outputXML); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Wrote %s and %s\n", outputJSON, outputXML)

	fmt.Println("\n=== Round-trip check: JSON -> struct -> compare to original ===")
	jsonBytes, _ := os.ReadFile(outputJSON)
	var reloaded Catalog
	if err := json.Unmarshal(jsonBytes, &reloaded); err != nil {
		fmt.Println("Error:", err)
		return
	}
	allMatch := true
	for i, p := range reloaded.Products {
		if p.Price != products[i].Price {
			allMatch = false
			fmt.Printf("  MISMATCH on %s: original %s, reloaded %s\n", p.SKU, products[i].Price, p.Price)
		}
	}
	if allMatch {
		fmt.Println("  every price round-tripped through JSON exactly — Marshal/Unmarshal agree")
	}

	fmt.Println("\n=== Round-trip check: XML -> struct -> compare to original ===")
	xmlBytes, _ := os.ReadFile(outputXML)
	var reloadedXML Catalog
	if err := xml.Unmarshal(xmlBytes, &reloadedXML); err != nil {
		fmt.Println("Error:", err)
		return
	}
	allMatch = true
	for i, p := range reloadedXML.Products {
		if p.Price != products[i].Price {
			allMatch = false
			fmt.Printf("  MISMATCH on %s: original %s, reloaded %s\n", p.SKU, products[i].Price, p.Price)
		}
	}
	if allMatch {
		fmt.Println("  every price round-tripped through XML exactly — Marshal/Unmarshal agree")
	}
}
