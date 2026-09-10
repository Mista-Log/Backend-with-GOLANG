// bench_test.go runs the SAME work through both packages, so the only
// variable is the implementation — run with:
//
//	go test -bench=. -benchmem ./...
//
// Expect the optimized versions to show meaningfully fewer allocs/op and
// bytes/op, with BuildCSVReport (fixing an O(n²) pattern) showing the
// widest gap as input size grows.
package main

import (
	"testing"

	"optimizerestapi/optimized"
	"optimizerestapi/shared"
	"optimizerestapi/unoptimized"
)

var products1k = shared.SampleProducts(1000)

func BenchmarkFilterByCategory_Unoptimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unoptimized.FilterByCategory(products1k, "electronics")
	}
}

func BenchmarkFilterByCategory_Optimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		optimized.FilterByCategory(products1k, "electronics")
	}
}

func BenchmarkBuildCSVReport_Unoptimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unoptimized.BuildCSVReport(products1k)
	}
}

func BenchmarkBuildCSVReport_Optimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		optimized.BuildCSVReport(products1k)
	}
}

func BenchmarkSearchByNamePattern_Unoptimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unoptimized.SearchByNamePattern(products1k, "Product 1.*")
	}
}

func BenchmarkSearchByNamePattern_Optimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		optimized.SearchByNamePattern(products1k, "Product 1.*")
	}
}

func BenchmarkSummarize_Unoptimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unoptimized.Summarize(products1k)
	}
}

func BenchmarkSummarize_Optimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		optimized.Summarize(products1k)
	}
}

func BenchmarkLogValue_Unoptimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unoptimized.LogValue(42.5)
	}
}

func BenchmarkLogValue_Optimized(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		optimized.LogValue(42.5)
	}
}
