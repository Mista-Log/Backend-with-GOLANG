// Package shared holds the Product type and sample data BOTH the
// unoptimized and optimized handlers use — so the benchmark comparison is
// strictly apples-to-apples: identical input, identical output shape,
// only the implementation differs.
package shared

import "fmt"

type Product struct {
	ID       int
	Name     string
	Category string
	Price    float64
	InStock  bool
}

func SampleProducts(n int) []Product {
	categories := []string{"electronics", "furniture", "books", "clothing", "toys"}
	products := make([]Product, n)
	for i := 0; i < n; i++ {
		products[i] = Product{
			ID:       i + 1,
			Name:     fmt.Sprintf("Product %d", i+1),
			Category: categories[i%len(categories)],
			Price:    float64(i%500) + 9.99,
			InStock:  i%3 != 0,
		}
	}
	return products
}
