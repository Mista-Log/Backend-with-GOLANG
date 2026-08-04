// Project 3: Book Library
//
// Unlike the other two projects (map-based), this one stores its catalog in
// a SLICE on purpose, so append/capacity/copy behavior — the Advanced
// section of this module — is directly visible and inspectable as you use
// the program.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Author struct {
	Name    string
	Country string
}

type Book struct {
	ISBN      string
	Title     string
	Author    Author
	Copies    int
	Available int
}

type Library struct {
	books       []Book // slice, not a map — see the capacity case study below
	checkoutLog []string
}

// findIndex does a linear scan — the trade-off for using a slice instead of
// a map: O(n) lookup by ISBN instead of O(1). Fine at library scale, and it
// keeps append/capacity behavior front and center for this project.
func (lib *Library) findIndex(isbn string) int {
	for i, b := range lib.books {
		if b.ISBN == isbn {
			return i
		}
	}
	return -1
}

func (lib *Library) addBook(b Book) error {
	if lib.findIndex(b.ISBN) != -1 {
		return fmt.Errorf("ISBN %q already exists", b.ISBN)
	}
	b.Available = b.Copies
	lib.books = append(lib.books, b) // this is the append that may grow capacity
	return nil
}

func (lib *Library) checkout(isbn string) error {
	i := lib.findIndex(isbn)
	if i == -1 {
		return fmt.Errorf("no book with ISBN %q", isbn)
	}
	if lib.books[i].Available <= 0 {
		return fmt.Errorf("no copies of %q currently available", lib.books[i].Title)
	}
	lib.books[i].Available--
	lib.checkoutLog = append(lib.checkoutLog, fmt.Sprintf("checked out: %s", lib.books[i].Title))
	return nil
}

func (lib *Library) returnBook(isbn string) error {
	i := lib.findIndex(isbn)
	if i == -1 {
		return fmt.Errorf("no book with ISBN %q", isbn)
	}
	if lib.books[i].Available >= lib.books[i].Copies {
		return fmt.Errorf("all copies of %q are already checked in", lib.books[i].Title)
	}
	lib.books[i].Available++
	lib.checkoutLog = append(lib.checkoutLog, fmt.Sprintf("returned: %s", lib.books[i].Title))
	return nil
}

// snapshot uses copy() to produce an INDEPENDENT slice of the current
// catalog state — unlike `frozen := lib.books`, which would only copy the
// slice header (pointer+len+cap) and still share the same underlying array.
func (lib *Library) snapshot() []Book {
	frozen := make([]Book, len(lib.books))
	n := copy(frozen, lib.books)
	_ = n // copy() returns how many elements it actually copied
	return frozen
}

func (lib *Library) printCatalog() {
	if len(lib.books) == 0 {
		fmt.Println("Catalog is empty.")
		return
	}
	for _, b := range lib.books {
		fmt.Printf("  [%s] %-25s by %-15s avail:%d/%d\n",
			b.ISBN, b.Title, b.Author.Name, b.Available, b.Copies)
	}
}

// printCapacityInfo makes append's reallocation behavior visible: watch cap
// jump (not grow by exactly 1) as you add books past the current capacity.
func (lib *Library) printCapacityInfo() {
	fmt.Printf("books slice: len=%d cap=%d\n", len(lib.books), cap(lib.books))
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	lib := &Library{}

menu:
	for {
		fmt.Println("\n1) Add book  2) Checkout  3) Return  4) Catalog  5) Slice capacity info  6) Checkout log  7) Snapshot demo  8) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			isbn := readLine(reader, "ISBN: ")
			title := readLine(reader, "Title: ")
			authorName := readLine(reader, "Author name: ")
			authorCountry := readLine(reader, "Author country: ")
			copies, err := strconv.Atoi(readLine(reader, "Number of copies: "))
			if err != nil {
				fmt.Println("Invalid number.")
				continue menu
			}
			book := Book{
				ISBN:   isbn,
				Title:  title,
				Author: Author{Name: authorName, Country: authorCountry},
				Copies: copies,
			}
			if err := lib.addBook(book); err != nil {
				fmt.Println("Error:", err)
			} else {
				lib.printCapacityInfo() // see capacity change right after the append
			}

		case "2":
			isbn := readLine(reader, "ISBN to check out: ")
			if err := lib.checkout(isbn); err != nil {
				fmt.Println("Error:", err)
			}

		case "3":
			isbn := readLine(reader, "ISBN to return: ")
			if err := lib.returnBook(isbn); err != nil {
				fmt.Println("Error:", err)
			}

		case "4":
			lib.printCatalog()

		case "5":
			lib.printCapacityInfo()

		case "6":
			if len(lib.checkoutLog) == 0 {
				fmt.Println("No activity yet.")
			}
			for _, entry := range lib.checkoutLog {
				fmt.Println(" -", entry)
			}

		case "7":
			frozen := lib.snapshot()
			fmt.Printf("Snapshot taken (%d books). Now checking out a book to prove independence...\n", len(frozen))
			if len(lib.books) > 0 {
				lib.books[0].Available = -999 // deliberately corrupt the LIVE data
				fmt.Printf("Live catalog books[0].Available is now: %d\n", lib.books[0].Available)
				fmt.Printf("Snapshot's books[0].Available is still: %d (unaffected — separate array)\n", frozen[0].Available)
			}

		case "8":
			break menu

		default:
			fmt.Println("Unknown option.")
		}
	}

	fmt.Println("\nGoodbye!")
}
