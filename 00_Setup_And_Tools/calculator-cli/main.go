// Project 2: Calculator CLI
//
// Two modes:
//   One-shot:  go run main.go -op add -a 4 -b 5
//   REPL:      go run main.go            (then type: add 4 5)
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrDivideByZero is a sentinel error — a package-level error value that callers
// can check against with errors.Is(). This is an idiomatic Go pattern.
var ErrDivideByZero = errors.New("division by zero")

// calculate holds all the arithmetic logic in one place so both the flag-based
// mode and the REPL mode can reuse it.
func calculate(op string, a, b float64) (float64, error) {
	switch op {
	case "add":
		return a + b, nil
	case "sub":
		return a - b, nil
	case "mul":
		return a * b, nil
	case "div":
		if b == 0 {
			return 0, ErrDivideByZero
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unknown operation %q (use add, sub, mul, div)", op)
	}
}

// runOneShot handles the `-op -a -b` flag-driven mode.
func runOneShot(op string, a, b float64) {
	result, err := calculate(op, a, b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Printf("%.2f %s %.2f = %.2f\n", a, opSymbol(op), b, result)
}

func opSymbol(op string) string {
	switch op {
	case "add":
		return "+"
	case "sub":
		return "-"
	case "mul":
		return "*"
	case "div":
		return "/"
	default:
		return "?"
	}
}

// runREPL is an interactive loop: read a line, parse it, calculate, print, repeat.
// This is the same read-eval-print pattern used by python's REPL, node's REPL, etc.
func runREPL() {
	fmt.Println("Go Calculator REPL — type e.g. `add 4 5`, or `exit` to quit")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // EOF (Ctrl+D) — exit cleanly
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Println("bye!")
			return
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			fmt.Println("expected: <op> <a> <b>, e.g. `add 4 5`")
			continue
		}

		a, errA := strconv.ParseFloat(parts[1], 64)
		b, errB := strconv.ParseFloat(parts[2], 64)
		if errA != nil || errB != nil {
			fmt.Println("both operands must be numbers")
			continue
		}

		result, err := calculate(parts[0], a, b)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Printf("= %.2f\n", result)
	}
}

func main() {
	op := flag.String("op", "", "operation: add, sub, mul, div")
	a := flag.Float64("a", 0, "first operand")
	b := flag.Float64("b", 0, "second operand")
	flag.Parse()

	if *op == "" {
		// No flags given — drop into interactive mode.
		runREPL()
		return
	}

	runOneShot(*op, *a, *b)
}
