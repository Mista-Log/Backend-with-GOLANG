// Project 1: ATM CLI
//
// Simulates a single-account ATM: PIN entry (3 attempts, labeled break to
// escape a nested retry loop), then a menu loop (for + switch) to check
// balance, deposit, withdraw, or exit. defer logs a session summary no
// matter which path ends the session.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const correctPIN = "1234"

type account struct {
	balance float64
}

func (a *account) deposit(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}
	a.balance += amount
	return nil
}

func (a *account) withdraw(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive")
	}
	if amount > a.balance {
		return fmt.Errorf("insufficient funds — balance is %.2f", a.balance)
	}
	a.balance -= amount
	return nil
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// authenticate gives the user 3 attempts, using a LABELED loop so a single
// `break attempts` can escape the retry loop cleanly from inside the
// if-statement that checks the PIN.
func authenticate(reader *bufio.Reader) bool {
	const maxAttempts = 3
	authenticated := false

attempts:
	for i := 1; i <= maxAttempts; i++ {
		pin := readLine(reader, fmt.Sprintf("Enter PIN (attempt %d/%d): ", i, maxAttempts))
		switch pin {
		case correctPIN:
			authenticated = true
			break attempts // exits the labeled loop immediately on success
		default:
			fmt.Println("Incorrect PIN.")
			if i == maxAttempts {
				fmt.Println("Too many failed attempts. Card retained.")
			}
			continue attempts // explicit, though the loop would continue anyway
		}
	}
	return authenticated
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	acc := &account{balance: 500}

	// defer here runs no matter which menu option ends the session —
	// success, insufficient funds along the way, or explicit exit.
	defer func() {
		fmt.Printf("\nSession ended. Final balance: %.2f. Thank you.\n", acc.balance)
	}()

	if !authenticate(reader) {
		os.Exit(1)
	}
	fmt.Println("\nAuthenticated. Welcome!")

menu:
	for {
		fmt.Println("\n1) Check balance  2) Deposit  3) Withdraw  4) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			fmt.Printf("Balance: %.2f\n", acc.balance)

		case "2":
			amtStr := readLine(reader, "Amount to deposit: ")
			amount, err := strconv.ParseFloat(amtStr, 64)
			if err != nil {
				fmt.Println("Invalid amount.")
				continue menu
			}
			if err := acc.deposit(amount); err != nil {
				fmt.Println("Error:", err)
				continue menu
			}
			fmt.Printf("Deposited %.2f. New balance: %.2f\n", amount, acc.balance)

		case "3":
			amtStr := readLine(reader, "Amount to withdraw: ")
			amount, err := strconv.ParseFloat(amtStr, 64)
			if err != nil {
				fmt.Println("Invalid amount.")
				continue menu
			}
			if err := acc.withdraw(amount); err != nil {
				fmt.Println("Error:", err)
				continue menu
			}
			fmt.Printf("Withdrew %.2f. New balance: %.2f\n", amount, acc.balance)

		case "4":
			break menu // labeled break — exits the menu `for`, not just the switch

		default:
			fmt.Println("Unknown option, try again.")
		}
	}
}
