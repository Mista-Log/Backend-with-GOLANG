package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	
	// Test variables
	name := "Go Developer"
	age := 25
	
	fmt.Printf("Name: %s, Age: %d\n", name, age)
	
	// Test slice
	numbers := []int{1, 2, 3, 4, 5}
	for i, num := range numbers {
		fmt.Printf("Index %d: %d\n", i, num)
	}
	
	// Test function call
	result := add(10, 20)
	fmt.Printf("Sum: %d\n", result)
}

func add(a, b int) int {
	return a + b
}
