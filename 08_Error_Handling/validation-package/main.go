// Demo entry point for the validation package — run with: go run .
package main

import (
	"errors"
	"fmt"

	"validationdemo/validation"
)

type SignupForm struct {
	Name  string
	Email string
	Age   int
}

// ValidateSignup runs every check unconditionally (each validator call
// happens immediately, before All ever sees the results) and merges
// whatever failed into one error via validation.All.
func ValidateSignup(f SignupForm) error {
	return validation.All(
		validation.Required("Name", f.Name),
		validation.MinLength("Name", f.Name, 2),
		validation.MaxLength("Name", f.Name, 50),
		validation.Required("Email", f.Email),
		validation.Email("Email", f.Email),
		validation.Range("Age", f.Age, 13, 120),
	)
}

func main() {
	fmt.Println("=== Valid form ===")
	err := ValidateSignup(SignupForm{Name: "Ada", Email: "ada@example.com", Age: 28})
	fmt.Println("error:", err) // nil

	fmt.Println("\n=== Invalid form (multiple failures at once) ===")
	err = ValidateSignup(SignupForm{Name: "A", Email: "not-an-email", Age: 5})
	fmt.Println("joined error message:")
	fmt.Println(err)

	fmt.Println("\n=== errors.Is — did AGE specifically fail as out-of-range? ===")
	fmt.Println(errors.Is(err, validation.ErrOutOfRange)) // true, even though err is really 3 merged errors

	fmt.Println("\n=== errors.Is — did a completely unrelated failure happen? ===")
	fmt.Println(errors.Is(err, validation.ErrTooLong)) // false — Name was too SHORT, not too long

	fmt.Println("\n=== errors.As — pull out ONE FieldError's structured data ===")
	var fieldErr *validation.FieldError
	if errors.As(err, &fieldErr) {
		fmt.Printf("first matching field error -> Field: %s, Err: %v\n", fieldErr.Field, fieldErr.Err)
	}

	fmt.Println("\n=== validation.Split — every individual failure, for a field-by-field report ===")
	for _, e := range validation.Split(err) {
		var fe *validation.FieldError
		if errors.As(e, &fe) {
			fmt.Printf("  %-6s -> %v\n", fe.Field, fe.Err)
		}
	}
}
