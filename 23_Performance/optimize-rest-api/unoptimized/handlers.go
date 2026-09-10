// Package unoptimized implements a product-search/report set of
// functions with EVERY inefficiency the guide's Memory Optimization
// section warns about, deliberately, so the optimized package (and the
// benchmarks comparing them) has something real to fix.
package unoptimized

import (
	"fmt"
	"regexp"
	"strings"

	"optimizerestapi/shared"
)

// FilterByCategory: inefficiency #1 — no capacity pre-allocation.
// append() here may reallocate and copy the growing slice multiple times
// as it doubles in capacity (Module 04's reallocation behavior), entirely
// avoidable since we could estimate a reasonable starting size.
func FilterByCategory(products []shared.Product, category string) []shared.Product {
	var result []shared.Product
	for _, p := range products {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// BuildCSVReport: inefficiency #2 — string concatenation via + in a loop.
// Each += allocates an entirely NEW string and copies everything
// accumulated so far into it — O(n squared) total work for n rows,
// exactly the guide's buildCSV anti-example.
func BuildCSVReport(products []shared.Product) string {
	result := "id,name,category,price,inStock\n"
	for _, p := range products {
		result += fmt.Sprintf("%d,%s,%s,%.2f,%t\n", p.ID, p.Name, p.Category, p.Price, p.InStock)
	}
	return result
}

// SearchByNamePattern: inefficiency #3 — recompiling the SAME regular
// expression on every single call, instead of once. regexp.Compile does
// real, non-trivial work parsing the pattern into a matching engine —
// paying that cost per-search instead of once, ever, is pure waste.
func SearchByNamePattern(products []shared.Product, pattern string) ([]shared.Product, error) {
	re, err := regexp.Compile(pattern) // recompiled EVERY call
	if err != nil {
		return nil, err
	}
	var matches []shared.Product // also missing a capacity hint (inefficiency #1 again)
	for _, p := range products {
		if re.MatchString(p.Name) {
			matches = append(matches, p)
		}
	}
	return matches, nil
}

// Summarize: inefficiency #4 — an unnecessary intermediate []string plus
// strings.Join, when a direct strings.Builder loop (as the guide's
// buildCSV fix shows) would avoid the extra slice entirely.
func Summarize(products []shared.Product) string {
	var lines []string
	for _, p := range products {
		lines = append(lines, fmt.Sprintf("%s: $%.2f", p.Name, p.Price))
	}
	return strings.Join(lines, "; ")
}

// LogValue: inefficiency #5 — an `any`-typed parameter on a HOT PATH
// function forces its argument to escape to the heap every call (the
// guide's Escape Analysis section's exact example), even though the
// caller only ever needs it for the duration of this one call.
func LogValue(v any) string {
	return fmt.Sprintf("%v", v)
}
