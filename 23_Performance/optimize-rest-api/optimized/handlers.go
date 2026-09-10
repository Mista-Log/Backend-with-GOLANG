// Package optimized fixes every inefficiency in the unoptimized package,
// one function at a time — same inputs, same outputs, measurably less
// work per call. See bench_test.go for the head-to-head numbers.
package optimized

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"optimizerestapi/shared"
)

// FilterByCategory: FIX — pre-allocate a reasonable capacity. We can't
// know the exact match count without scanning once, but len(products)/5
// is a defensible estimate for "one category among several" — good
// enough to avoid most reallocations without a wasted second pass.
func FilterByCategory(products []shared.Product, category string) []shared.Product {
	result := make([]shared.Product, 0, len(products)/5+1)
	for _, p := range products {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// BuildCSVReport: FIX — strings.Builder with a pre-sized buffer, per the
// guide's exact fix. WriteString/WriteByte append into one growing
// internal buffer instead of allocating a new string on every row.
func BuildCSVReport(products []shared.Product) string {
	var b strings.Builder
	b.Grow(32 + len(products)*40) // header + a rough per-row estimate
	b.WriteString("id,name,category,price,inStock\n")
	for _, p := range products {
		fmt.Fprintf(&b, "%d,%s,%s,%.2f,%t\n", p.ID, p.Name, p.Category, p.Price, p.InStock)
	}
	return b.String()
}

// searchPatternCache holds compiled regexes, keyed by their source
// pattern — FIX for inefficiency #3: compile each DISTINCT pattern only
// ONCE, ever, no matter how many times it's searched for afterward.
// Mutex-protected (Module 12) since this cache is shared across
// concurrent request-handling goroutines.
var (
	patternCacheMu sync.Mutex
	patternCache   = make(map[string]*regexp.Regexp)
)

func compiledPattern(pattern string) (*regexp.Regexp, error) {
	patternCacheMu.Lock()
	defer patternCacheMu.Unlock()
	if re, ok := patternCache[pattern]; ok {
		return re, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	patternCache[pattern] = re
	return re, nil
}

func SearchByNamePattern(products []shared.Product, pattern string) ([]shared.Product, error) {
	re, err := compiledPattern(pattern)
	if err != nil {
		return nil, err
	}
	matches := make([]shared.Product, 0, len(products)/10+1) // FIX for inefficiency #1, again
	for _, p := range products {
		if re.MatchString(p.Name) {
			matches = append(matches, p)
		}
	}
	return matches, nil
}

// Summarize: FIX — build directly into a strings.Builder, skipping the
// intermediate []string and strings.Join entirely.
func Summarize(products []shared.Product) string {
	var b strings.Builder
	b.Grow(len(products) * 20)
	for i, p := range products {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s: $%.2f", p.Name, p.Price)
	}
	return b.String()
}

// LogValue: FIX — a concrete, typed parameter instead of `any`. The
// caller's value can now stay on the STACK (Module 05) when the compiler
// can prove it doesn't need to escape, instead of being forced to the
// heap purely because of an interface-typed parameter.
func LogValue(v float64) string {
	return fmt.Sprintf("%v", v)
}
