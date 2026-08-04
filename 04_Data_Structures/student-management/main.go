// Project 2: Student Management
//
// Tracks students by ID. Demonstrates a NESTED struct (Address, accessed
// through its field name), EMBEDDING (GraduateStudent embeds Student), and
// slices used to hold each student's list of grades.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Address is NESTED inside Student (via a named field) — a clean "has-a"
// relationship, always accessed as student.Address.City.
type Address struct {
	City  string
	State string
}

type Student struct {
	ID      int
	Name    string
	Address Address
	Grades  []float64
}

// AddGrade appends to the Grades slice — a pointer receiver, since it needs
// to modify the actual stored Student, not a copy of it.
func (s *Student) AddGrade(grade float64) {
	s.Grades = append(s.Grades, grade)
}

// Average folds the Grades slice down to a single number — the same
// "Reduce" shape from Module 03's mathlib, written as a method this time.
func (s Student) Average() float64 {
	if len(s.Grades) == 0 {
		return 0
	}
	total := 0.0
	for _, g := range s.Grades {
		total += g
	}
	return total / float64(len(s.Grades))
}

// GraduateStudent EMBEDS Student — every field and method of Student is
// promoted onto GraduateStudent, which adds its own thesis-related fields.
type GraduateStudent struct {
	Student
	ThesisTitle string
	Advisor     string
}

type school struct {
	students   map[int]*Student
	graduates  map[int]*GraduateStudent
	nextID     int
}

func newSchool() *school {
	return &school{
		students:  make(map[int]*Student),
		graduates: make(map[int]*GraduateStudent),
		nextID:    1,
	}
}

func (sc *school) addStudent(name string, addr Address) int {
	id := sc.nextID
	sc.nextID++
	sc.students[id] = &Student{ID: id, Name: name, Address: addr}
	return id
}

func (sc *school) addGraduateStudent(name string, addr Address, thesis, advisor string) int {
	id := sc.nextID
	sc.nextID++
	sc.graduates[id] = &GraduateStudent{
		Student:     Student{ID: id, Name: name, Address: addr},
		ThesisTitle: thesis,
		Advisor:     advisor,
	}
	return id
}

// addGrade checks both maps, same pattern as the Inventory System project.
func (sc *school) addGrade(id int, grade float64) error {
	if s, ok := sc.students[id]; ok {
		s.AddGrade(grade)
		return nil
	}
	if g, ok := sc.graduates[id]; ok {
		g.AddGrade(grade) // PROMOTED method — GraduateStudent has no AddGrade of its own
		return nil
	}
	return fmt.Errorf("no student with ID %d", id)
}

// classAverage computes the average of averages across every student —
// ranges over both maps, using each Student's own Average() method.
func (sc *school) classAverage() float64 {
	var total float64
	var count int
	for _, s := range sc.students {
		total += s.Average()
		count++
	}
	for _, g := range sc.graduates {
		total += g.Average() // PROMOTED
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (sc *school) printAll() {
	var ids []int
	for id := range sc.students {
		ids = append(ids, id)
	}
	for id := range sc.graduates {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	if len(ids) == 0 {
		fmt.Println("No students yet.")
		return
	}

	for _, id := range ids {
		if s, ok := sc.students[id]; ok {
			fmt.Printf("  [%d] %-15s %s, %-12s avg:%.2f\n",
				s.ID, s.Name, s.Address.City, s.Address.State, s.Average())
			continue
		}
		if g, ok := sc.graduates[id]; ok {
			fmt.Printf("  [%d] %-15s %s, %-12s avg:%.2f  (Grad — thesis: %q, advisor: %s)\n",
				g.ID, g.Name, g.Address.City, g.Address.State, g.Average(), g.ThesisTitle, g.Advisor)
		}
	}
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	sc := newSchool()

menu:
	for {
		fmt.Println("\n1) Add student  2) Add graduate student  3) Add grade  4) List all  5) Class average  6) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			name := readLine(reader, "Name: ")
			city := readLine(reader, "City: ")
			state := readLine(reader, "State: ")
			id := sc.addStudent(name, Address{City: city, State: state})
			fmt.Printf("Added student with ID %d\n", id)

		case "2":
			name := readLine(reader, "Name: ")
			city := readLine(reader, "City: ")
			state := readLine(reader, "State: ")
			thesis := readLine(reader, "Thesis title: ")
			advisor := readLine(reader, "Advisor: ")
			id := sc.addGraduateStudent(name, Address{City: city, State: state}, thesis, advisor)
			fmt.Printf("Added graduate student with ID %d\n", id)

		case "3":
			id, errID := strconv.Atoi(readLine(reader, "Student ID: "))
			grade, errG := strconv.ParseFloat(readLine(reader, "Grade: "), 64)
			if errID != nil || errG != nil {
				fmt.Println("Invalid input.")
				continue menu
			}
			if err := sc.addGrade(id, grade); err != nil {
				fmt.Println("Error:", err)
			}

		case "4":
			sc.printAll()

		case "5":
			fmt.Printf("Class average: %.2f\n", sc.classAverage())

		case "6":
			break menu

		default:
			fmt.Println("Unknown option.")
		}
	}

	fmt.Println("\nGoodbye!")
}
