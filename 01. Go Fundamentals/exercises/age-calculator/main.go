// Exercise 2: Age Calculator
//
// Computes exact age in years, months, and days from a birth date.
// Usage: go run main.go -year 1997 -month 8 -day 21
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// calculateAge does manual calendar arithmetic rather than just subtracting
// durations, because "years/months/days between two dates" isn't a fixed
// number of hours (months have different lengths, leap years exist) — this
// is a good early lesson in why naive time-math is a common source of bugs.
func calculateAge(birth, now time.Time) (years, months, days int) {
	years = now.Year() - birth.Year()
	months = int(now.Month()) - int(birth.Month())
	days = now.Day() - birth.Day()

	// Borrow a "month" worth of days if the day count went negative —
	// same idea as borrowing in manual subtraction (e.g. 32 - 45 in base 10).
	if days < 0 {
		months--
		// Days in the month BEFORE `now`'s current month.
		prevMonth := now.AddDate(0, -1, 0)
		daysInPrevMonth := time.Date(prevMonth.Year(), prevMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		days += daysInPrevMonth
	}

	// Borrow a "year" worth of months if the month count went negative.
	if months < 0 {
		years--
		months += 12
	}

	return years, months, days
}

func main() {
	year := flag.Int("year", 0, "birth year, e.g. 1997")
	month := flag.Int("month", 0, "birth month (1-12)")
	day := flag.Int("day", 0, "birth day of month")
	flag.Parse()

	if *year == 0 || *month == 0 || *day == 0 {
		fmt.Fprintln(os.Stderr, "Error: -year, -month, and -day are all required")
		os.Exit(1)
	}

	birth := time.Date(*year, time.Month(*month), *day, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()

	if birth.After(now) {
		fmt.Fprintln(os.Stderr, "Error: birth date is in the future")
		os.Exit(1)
	}

	years, months, days := calculateAge(birth, now)
	fmt.Printf("Born:  %s\n", birth.Format("January 2, 2006"))
	fmt.Printf("Today: %s\n", now.Format("January 2, 2006"))
	fmt.Printf("Age:   %d years, %d months, %d days\n", years, months, days)

	// Bonus: total days lived, using straightforward duration subtraction —
	// this one IS just a fixed number of hours, so simple math is fine here.
	totalDays := int(now.Sub(birth).Hours() / 24)
	fmt.Printf("Total days lived: %d\n", totalDays)
}
