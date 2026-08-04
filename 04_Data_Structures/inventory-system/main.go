// Project 1: Inventory System
//
// Manages products in a map keyed by SKU for O(1) lookup. Demonstrates
// structs, embedding (Perishable embeds Product), maps of structs, and
// slices (for the low-stock report).
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Product is the base struct every inventory item shares.
type Product struct {
	SKU      string
	Name     string
	Price    float64
	Quantity int
}

// Perishable EMBEDS Product — it "has all of Product's fields promoted",
// plus its own ExpiryDate. This models "a Perishable IS-A Product, with
// extra data" using composition instead of class inheritance.
type Perishable struct {
	Product
	ExpiryDate time.Time
}

// IsExpired is a method only Perishable has — Product itself knows nothing
// about expiry dates. Embedding doesn't give Product any of Perishable's
// behavior; it only flows one direction (embedded -> outer).
func (p Perishable) IsExpired() bool {
	return time.Now().After(p.ExpiryDate)
}

// inventory holds regular products by SKU, and perishables separately —
// two maps because Perishable and Product are different types, even though
// Perishable embeds Product.
type inventory struct {
	products    map[string]*Product
	perishables map[string]*Perishable
}

func newInventory() *inventory {
	return &inventory{
		products:    make(map[string]*Product),
		perishables: make(map[string]*Perishable),
	}
}

func (inv *inventory) addProduct(p Product) error {
	if _, exists := inv.products[p.SKU]; exists {
		return fmt.Errorf("SKU %q already exists", p.SKU)
	}
	inv.products[p.SKU] = &p
	return nil
}

func (inv *inventory) addPerishable(p Perishable) error {
	if _, exists := inv.perishables[p.SKU]; exists {
		return fmt.Errorf("SKU %q already exists", p.SKU)
	}
	inv.perishables[p.SKU] = &p
	return nil
}

// adjustQuantity works on EITHER map, since a positive delta restocks and a
// negative one sells — checking both maps keeps the menu logic simple.
func (inv *inventory) adjustQuantity(sku string, delta int) error {
	if p, ok := inv.products[sku]; ok {
		if p.Quantity+delta < 0 {
			return fmt.Errorf("cannot reduce %s below zero (have %d)", sku, p.Quantity)
		}
		p.Quantity += delta
		return nil
	}
	if p, ok := inv.perishables[sku]; ok {
		if p.Quantity+delta < 0 {
			return fmt.Errorf("cannot reduce %s below zero (have %d)", sku, p.Quantity)
		}
		p.Quantity += delta
		return nil
	}
	return fmt.Errorf("no product with SKU %q", sku)
}

// totalValue sums Price*Quantity across BOTH maps — a running total built
// by ranging over each map in turn.
func (inv *inventory) totalValue() float64 {
	total := 0.0
	for _, p := range inv.products {
		total += p.Price * float64(p.Quantity)
	}
	for _, p := range inv.perishables {
		total += p.Price * float64(p.Quantity)
	}
	return total
}

// lowStock returns SKUs (sorted) with quantity at or below the threshold —
// built as a []string so the caller gets a stable, orderable result, unlike
// ranging over the maps directly.
func (inv *inventory) lowStock(threshold int) []string {
	var skus []string
	for sku, p := range inv.products {
		if p.Quantity <= threshold {
			skus = append(skus, sku)
		}
	}
	for sku, p := range inv.perishables {
		if p.Quantity <= threshold {
			skus = append(skus, sku)
		}
	}
	sort.Strings(skus)
	return skus
}

func (inv *inventory) printAll() {
	var skus []string
	for sku := range inv.products {
		skus = append(skus, sku)
	}
	for sku := range inv.perishables {
		skus = append(skus, sku)
	}
	sort.Strings(skus)

	if len(skus) == 0 {
		fmt.Println("Inventory is empty.")
		return
	}

	for _, sku := range skus {
		if p, ok := inv.products[sku]; ok {
			fmt.Printf("  [%s] %-15s qty:%-4d price:%.2f\n", p.SKU, p.Name, p.Quantity, p.Price)
			continue
		}
		if p, ok := inv.perishables[sku]; ok {
			expired := ""
			if p.IsExpired() {
				expired = " (EXPIRED)"
			}
			fmt.Printf("  [%s] %-15s qty:%-4d price:%.2f expires:%s%s\n",
				p.SKU, p.Name, p.Quantity, p.Price, p.ExpiryDate.Format("2006-01-02"), expired)
		}
	}
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	inv := newInventory()

menu:
	for {
		fmt.Println("\n1) Add product  2) Add perishable  3) List all  4) Adjust quantity  5) Low stock report  6) Total value  7) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			sku := readLine(reader, "SKU: ")
			name := readLine(reader, "Name: ")
			price, errP := strconv.ParseFloat(readLine(reader, "Price: "), 64)
			qty, errQ := strconv.Atoi(readLine(reader, "Quantity: "))
			if errP != nil || errQ != nil {
				fmt.Println("Invalid price or quantity.")
				continue menu
			}
			if err := inv.addProduct(Product{SKU: sku, Name: name, Price: price, Quantity: qty}); err != nil {
				fmt.Println("Error:", err)
			}

		case "2":
			sku := readLine(reader, "SKU: ")
			name := readLine(reader, "Name: ")
			price, errP := strconv.ParseFloat(readLine(reader, "Price: "), 64)
			qty, errQ := strconv.Atoi(readLine(reader, "Quantity: "))
			daysUntilExpiry, errD := strconv.Atoi(readLine(reader, "Days until expiry: "))
			if errP != nil || errQ != nil || errD != nil {
				fmt.Println("Invalid input.")
				continue menu
			}
			p := Perishable{
				Product:    Product{SKU: sku, Name: name, Price: price, Quantity: qty},
				ExpiryDate: time.Now().AddDate(0, 0, daysUntilExpiry),
			}
			if err := inv.addPerishable(p); err != nil {
				fmt.Println("Error:", err)
			}

		case "3":
			inv.printAll()

		case "4":
			sku := readLine(reader, "SKU: ")
			delta, err := strconv.Atoi(readLine(reader, "Change in quantity (negative to sell): "))
			if err != nil {
				fmt.Println("Invalid number.")
				continue menu
			}
			if err := inv.adjustQuantity(sku, delta); err != nil {
				fmt.Println("Error:", err)
			}

		case "5":
			threshold, err := strconv.Atoi(readLine(reader, "Low stock threshold: "))
			if err != nil {
				fmt.Println("Invalid number.")
				continue menu
			}
			skus := inv.lowStock(threshold)
			if len(skus) == 0 {
				fmt.Println("Nothing at or below that threshold.")
			}
			for _, sku := range skus {
				fmt.Println(" -", sku)
			}

		case "6":
			fmt.Printf("Total inventory value: %.2f\n", inv.totalValue())

		case "7":
			break menu

		default:
			fmt.Println("Unknown option.")
		}
	}

	fmt.Println("\nGoodbye!")
}
