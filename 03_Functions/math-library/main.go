// Demo entry point for the mathlib package — run with: go run .
package main

import (
	"fmt"

	"mathlibdemo/mathlib"
)

func main() {
	fmt.Println("== Factorial (recursion) ==")
	for _, n := range []int{0, 1, 5, 10} {
		result, err := mathlib.Factorial(n)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Printf("%d! = %d\n", n, result)
	}

	fmt.Println("\n== Fibonacci (memoized closure) ==")
	fib := mathlib.NewMemoizedFibonacci() // one closure instance, one shared cache
	for n := 0; n <= 10; n++ {
		fmt.Printf("fib(%d)=%d ", n, fib(n))
	}
	fmt.Println()

	fmt.Println("\n== IsPrime ==")
	for n := 2; n <= 20; n++ {
		if mathlib.IsPrime(n) {
			fmt.Printf("%d ", n)
		}
	}
	fmt.Println()

	fmt.Println("\n== GCD / LCM (recursion) ==")
	fmt.Printf("GCD(48, 18) = %d\n", mathlib.GCD(48, 18))
	fmt.Printf("LCM(4, 6)   = %d\n", mathlib.LCM(4, 6))

	fmt.Println("\n== Sum / Average (variadic + named returns) ==")
	fmt.Println("Sum(1,2,3,4,5) =", mathlib.Sum(1, 2, 3, 4, 5))
	avg, err := mathlib.Average(1, 2, 3, 4, 5)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Average(1,2,3,4,5) = %.2f\n", avg)
	}
	if _, err := mathlib.Average(); err != nil {
		fmt.Println("Average() with no args ->", err)
	}

	fmt.Println("\n== Map / Filter / Reduce (higher-order functions) ==")
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	doubled := mathlib.Map(nums, func(n int) int { return n * 2 })
	fmt.Println("doubled:", doubled)

	evens := mathlib.Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("evens:", evens)

	total := mathlib.Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Println("sum via Reduce:", total)

	product := mathlib.Reduce(nums, 1, func(acc, n int) int { return acc * n })
	fmt.Println("product via Reduce:", product)

	// Composing all three, closure-style, in one pipeline:
	sumOfSquaredEvens := mathlib.Reduce(
		mathlib.Map(
			mathlib.Filter(nums, func(n int) bool { return n%2 == 0 }),
			func(n int) int { return n * n },
		),
		0,
		func(acc, n int) int { return acc + n },
	)
	fmt.Println("sum of squared evens:", sumOfSquaredEvens)
}
