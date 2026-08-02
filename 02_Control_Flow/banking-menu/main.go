// Project 3: Banking Menu
//
// Manages multiple named accounts: create, list, deposit/withdraw/transfer,
// and delete. Two nested menu loops (main menu + per-account menu) show how
// labels let you control exactly which loop a break/continue affects.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type account struct {
	owner   string
	balance float64
}

// bank holds every account, keyed by owner name for easy lookup.
type bank struct {
	accounts map[string]*account
}

func newBank() *bank {
	return &bank{accounts: make(map[string]*account)}
}

func (b *bank) create(owner string, opening float64) error {
	if _, exists := b.accounts[owner]; exists {
		return fmt.Errorf("an account for %q already exists", owner)
	}
	if opening < 0 {
		return fmt.Errorf("opening balance cannot be negative")
	}
	b.accounts[owner] = &account{owner: owner, balance: opening}
	return nil
}

// sortedOwners returns owner names in alphabetical order — map iteration
// order in Go is intentionally randomized, so anything user-facing that
// needs a stable order has to sort explicitly, every time.
func (b *bank) sortedOwners() []string {
	owners := make([]string, 0, len(b.accounts))
	for owner := range b.accounts {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	return owners
}

func (b *bank) transfer(from, to string, amount float64) error {
	src, ok := b.accounts[from]
	if !ok {
		return fmt.Errorf("no account for %q", from)
	}
	dst, ok := b.accounts[to]
	if !ok {
		return fmt.Errorf("no account for %q", to)
	}
	if amount <= 0 {
		return fmt.Errorf("transfer amount must be positive")
	}
	if amount > src.balance {
		return fmt.Errorf("insufficient funds in %q's account", from)
	}
	src.balance -= amount
	dst.balance += amount
	return nil
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func readFloat(reader *bufio.Reader, prompt string) (float64, error) {
	return strconv.ParseFloat(readLine(reader, prompt), 64)
}

// manageAccount is its own labeled loop for a single account's actions.
// "back" returns to the main menu; it does NOT exit the whole program —
// that distinction is exactly why this loop needs its own label, separate
// from the outer main-menu loop.
func manageAccount(reader *bufio.Reader, b *bank, owner string) {
account:
	for {
		acc := b.accounts[owner] // re-fetch each loop in case balance changed
		fmt.Printf("\n-- %s (balance: %.2f) --\n", acc.owner, acc.balance)
		fmt.Println("1) Deposit  2) Withdraw  3) Transfer out  4) Back to main menu")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			amount, err := readFloat(reader, "Deposit amount: ")
			if err != nil || amount <= 0 {
				fmt.Println("Invalid amount.")
				continue account
			}
			acc.balance += amount
			fmt.Printf("New balance: %.2f\n", acc.balance)

		case "2":
			amount, err := readFloat(reader, "Withdraw amount: ")
			if err != nil || amount <= 0 {
				fmt.Println("Invalid amount.")
				continue account
			}
			if amount > acc.balance {
				fmt.Println("Insufficient funds.")
				continue account
			}
			acc.balance -= amount
			fmt.Printf("New balance: %.2f\n", acc.balance)

		case "3":
			to := readLine(reader, "Transfer to (owner name): ")
			amount, err := readFloat(reader, "Amount: ")
			if err != nil {
				fmt.Println("Invalid amount.")
				continue account
			}
			if err := b.transfer(owner, to, amount); err != nil {
				fmt.Println("Error:", err)
				continue account
			}
			fmt.Printf("Transferred %.2f to %s.\n", amount, to)

		case "4":
			break account // returns control to the MAIN menu loop below

		default:
			fmt.Println("Unknown option.")
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	b := newBank()

main:
	for {
		fmt.Println("\n1) Create account  2) List accounts  3) Manage an account  4) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			owner := readLine(reader, "Owner name: ")
			opening, err := readFloat(reader, "Opening balance: ")
			if err != nil {
				fmt.Println("Invalid amount.")
				continue main
			}
			if err := b.create(owner, opening); err != nil {
				fmt.Println("Error:", err)
				continue main
			}
			fmt.Printf("Account created for %s.\n", owner)

		case "2":
			owners := b.sortedOwners()
			if len(owners) == 0 {
				fmt.Println("No accounts yet.")
				continue main
			}
			for _, owner := range owners {
				acc := b.accounts[owner]
				fmt.Printf("  %-15s balance: %.2f\n", acc.owner, acc.balance)
			}

		case "3":
			owner := readLine(reader, "Which account (owner name)? ")
			if _, ok := b.accounts[owner]; !ok {
				fmt.Println("No such account.")
				continue main
			}
			manageAccount(reader, b, owner)

		case "4":
			break main

		default:
			fmt.Println("Unknown option.")
		}
	}

	fmt.Println("\nGoodbye!")
}
