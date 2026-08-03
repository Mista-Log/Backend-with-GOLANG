// Package mathlib is a small math library demonstrating recursion, variadic
// functions, multiple/named returns, and higher-order functions.
package mathlib

import "fmt"

// Factorial uses straightforward recursion with a base case at n <= 1.
func Factorial(n int) (int, error) {
	if n < 0 {
		return 0, fmt.Errorf("factorial undefined for negative number %d", n)
	}
	if n <= 1 {
		return 1, nil
	}
	result, err := Factorial(n - 1)
	if err != nil {
		return 0, err
	}
	return n * result, nil
}

// Fibonacci returns the nth Fibonacci number (0-indexed: Fibonacci(0) == 0).
// Naive recursion here would be exponential time — instead this uses
// MEMOIZATION via a CLOSURE that captures a cache map, so repeated calls to
// the SAME returned function reuse previous work.
func NewMemoizedFibonacci() func(n int) int {
	cache := map[int]int{0: 0, 1: 1} // captured by the closure below

	var fib func(n int) int
	fib = func(n int) int {
		if v, ok := cache[n]; ok {
			return v
		}
		result := fib(n-1) + fib(n-2)
		cache[n] = result
		return result
	}
	return fib
}

// IsPrime reports whether n is a prime number, checking only up to sqrt(n)
// since any factor larger than that would have a matching factor smaller
// than it (already would have been found).
func IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// GCD computes the greatest common divisor via the recursive Euclidean
// algorithm: gcd(a, b) == gcd(b, a mod b), until b reaches 0.
func GCD(a, b int) int {
	if b == 0 {
		return a
	}
	return GCD(b, a%b)
}

// LCM computes the least common multiple, built directly on top of GCD —
// a good small example of composing one function from another.
func LCM(a, b int) int {
	return a / GCD(a, b) * b
}

// Sum is VARIADIC: call it with any number of ints, including zero.
func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Average uses a NAMED return so the "did the caller pass any numbers at
// all?" error case reads clearly at the call site.
func Average(nums ...int) (avg float64, err error) {
	if len(nums) == 0 {
		err = fmt.Errorf("cannot average zero numbers")
		return
	}
	avg = float64(Sum(nums...)) / float64(len(nums))
	return
}

// Map applies f to every element, returning a new slice — a classic
// HIGHER-ORDER FUNCTION: it takes a function as a parameter.
func Map(nums []int, f func(int) int) []int {
	result := make([]int, len(nums))
	for i, n := range nums {
		result[i] = f(n)
	}
	return result
}

// Filter keeps only the elements for which keep returns true.
func Filter(nums []int, keep func(int) bool) []int {
	var result []int
	for _, n := range nums {
		if keep(n) {
			result = append(result, n)
		}
	}
	return result
}

// Reduce folds a slice down to a single value, given a starting value and a
// combining function — this is the same shape as sum/product/max, generalized.
func Reduce(nums []int, initial int, f func(acc, n int) int) int {
	acc := initial
	for _, n := range nums {
		acc = f(acc, n)
	}
	return acc
}
