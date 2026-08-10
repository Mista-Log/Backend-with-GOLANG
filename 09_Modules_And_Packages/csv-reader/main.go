// Project 2: CSV Reader
//
// Generates a sample employee CSV, reads it as a STREAM (row by row, via
// encoding/csv wrapped around a bufio.Reader) rather than loading it all at
// once, filters rows matching a department, and writes the filtered result
// to a new CSV — through the same temp-file-then-rename safe-write pattern
// as the Log Parser project, plus an explicit permissions check on the
// output directory.
package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Employee struct {
	Name       string
	Department string
	Salary     float64
}

func generateSampleCSV(path string) error {
	content := `name,department,salary
Ada Lovelace,Engineering,95000
Grace Hopper,Engineering,102000
Kemi Adeyemi,Sales,68000
Tolu Bakare,Engineering,88000
Zainab Musa,Marketing,71000
Chidi Okafor,Sales,72000
Amara Chukwu,Engineering,110000
`
	return os.WriteFile(path, []byte(content), 0644)
}

// readEmployeesStreaming reads row by row using csv.Reader.Read() in a
// loop, instead of csv.Reader.ReadAll() — the STREAMING approach from the
// guide's section 6, so a very large CSV never needs to fit entirely in
// memory. Wrapping os.Open's *os.File in a bufio.Reader first means every
// underlying read is buffered too (section 7), not just line-by-line.
func readEmployeesStreaming(path string) ([]Employee, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bufReader := bufio.NewReader(f) // explicit buffering, per the guide
	reader := csv.NewReader(bufReader)

	header, err := reader.Read() // first row: column names
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	colIndex := indexColumns(header)

	var employees []Employee
	for {
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) { // encoding/csv returns io.EOF at the true end —
				break                     // this is the errors.Is habit from Module 08,
			}                              // more robust than comparing error strings
			return employees, fmt.Errorf("reading row: %w", err)
		}

		salary, err := strconv.ParseFloat(row[colIndex["salary"]], 64)
		if err != nil {
			return employees, fmt.Errorf("invalid salary in row %v: %w", row, err)
		}

		employees = append(employees, Employee{
			Name:       row[colIndex["name"]],
			Department: row[colIndex["department"]],
			Salary:     salary,
		})
	}
	return employees, nil
}

// indexColumns maps column NAME to column INDEX, so row access doesn't
// depend on a fixed column order — a more robust habit than row[0], row[1]...
func indexColumns(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, name := range header {
		idx[name] = i
	}
	return idx
}

func filterByDepartment(employees []Employee, department string) []Employee {
	var out []Employee
	for _, e := range employees {
		if e.Department == department {
			out = append(out, e)
		}
	}
	return out
}

// writeEmployeesCSV writes through the SAME safe-write pattern as the Log
// Parser project: temp file, buffered csv.Writer, explicit Flush, explicit
// permissions, then an atomic os.Rename into the real path.
func writeEmployeesCSV(path string, employees []Employee) error {
	outDir := filepath.Dir(path)
	// os.MkdirAll is a no-op if outDir already exists with compatible
	// permissions — explicit 0755 here regardless, per the guide's section
	// on permissions (owner: rwx, group/others: rx — standard for a
	// directory that needs to be enterable and listable).
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(outDir, "employees-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	bufWriter := bufio.NewWriter(tmp)
	csvWriter := csv.NewWriter(bufWriter)

	if err := csvWriter.Write([]string{"name", "department", "salary"}); err != nil {
		return err
	}
	for _, e := range employees {
		row := []string{e.Name, e.Department, strconv.FormatFloat(e.Salary, 'f', 2, 64)}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	csvWriter.Flush() // flushes csv.Writer's OWN small buffer into bufWriter
	if err := csvWriter.Error(); err != nil {
		return err
	}
	if err := bufWriter.Flush(); err != nil { // flushes bufWriter into the actual file
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func main() {
	const inputPath = "data/employees.csv"
	const outputPath = "data/engineering.csv"

	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Println("Error creating data directory:", err)
		return
	}
	if err := generateSampleCSV(inputPath); err != nil {
		fmt.Println("Error generating sample CSV:", err)
		return
	}

	employees, err := readEmployeesStreaming(inputPath)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	fmt.Printf("Read %d employees from %s:\n", len(employees), inputPath)
	total := 0.0
	for _, e := range employees {
		fmt.Printf("  %-15s %-12s $%.2f\n", e.Name, e.Department, e.Salary)
		total += e.Salary
	}
	fmt.Printf("Average salary: $%.2f\n", total/float64(len(employees)))

	engineering := filterByDepartment(employees, "Engineering")
	fmt.Printf("\nFiltered to %d Engineering employee(s)\n", len(engineering))

	if err := writeEmployeesCSV(outputPath, engineering); err != nil {
		fmt.Println("Error writing filtered CSV:", err)
		return
	}
	fmt.Printf("Filtered CSV written to %s\n", outputPath)

	info, _ := os.Stat(outputPath)
	fmt.Printf("Output file permissions: %v\n", info.Mode())
}
