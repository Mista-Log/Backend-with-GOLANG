// Exercise 1: Temperature Converter
//
// Converts between Celsius, Fahrenheit, and Kelvin.
// Usage: go run main.go -val 100 -from c -to f
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// celsiusToFahrenheit and friends are small, single-purpose conversion
// functions — this is the idiomatic Go way to keep each unit conversion
// testable in isolation.
func celsiusToFahrenheit(c float64) float64 { return c*9/5 + 32 }
func fahrenheitToCelsius(f float64) float64 { return (f - 32) * 5 / 9 }
func celsiusToKelvin(c float64) float64     { return c + 273.15 }
func kelvinToCelsius(k float64) float64     { return k - 273.15 }

// convert always routes through Celsius as a common "base unit" — this avoids
// writing 6 separate direct conversion formulas (c->f, f->c, c->k, k->c, f->k, k->f)
// and instead only needs 4, which is a pattern worth remembering for any
// "convert between N units" problem.
func convert(value float64, from, to string) (float64, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	// Step 1: normalize input to Celsius.
	var celsius float64
	switch from {
	case "c":
		celsius = value
	case "f":
		celsius = fahrenheitToCelsius(value)
	case "k":
		celsius = kelvinToCelsius(value)
	default:
		return 0, fmt.Errorf("unknown source unit %q (use c, f, or k)", from)
	}

	// Step 2: convert from Celsius to the target unit.
	switch to {
	case "c":
		return celsius, nil
	case "f":
		return celsiusToFahrenheit(celsius), nil
	case "k":
		return celsiusToKelvin(celsius), nil
	default:
		return 0, fmt.Errorf("unknown target unit %q (use c, f, or k)", to)
	}
}

func unitName(u string) string {
	switch strings.ToLower(u) {
	case "c":
		return "°C"
	case "f":
		return "°F"
	case "k":
		return "K"
	default:
		return u
	}
}

func main() {
	val := flag.Float64("val", 0, "temperature value to convert")
	from := flag.String("from", "c", "source unit: c, f, or k")
	to := flag.String("to", "f", "target unit: c, f, or k")
	flag.Parse()

	result, err := convert(*val, *from, *to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Printf("%.2f%s = %.2f%s\n", *val, unitName(*from), result, unitName(*to))
}
