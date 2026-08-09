// Package validation provides small, composable field validators. Every
// validator returns either nil or a *FieldError wrapping a SENTINEL error
// (ErrRequired, ErrTooShort, ...) — so callers can check WHICH KIND of
// failure happened with errors.Is, or pull out the failing field name with
// errors.As, even after multiple validators' errors have been merged
// together with errors.Join.
package validation

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors — checked by IDENTITY via errors.Is, not by comparing
// message strings, so wrapping (see FieldError below) never breaks them.
var (
	ErrRequired     = errors.New("value is required")
	ErrTooShort     = errors.New("value is too short")
	ErrTooLong      = errors.New("value is too long")
	ErrInvalidEmail = errors.New("invalid email format")
	ErrOutOfRange   = errors.New("value out of range")
)

// FieldError is a CUSTOM ERROR TYPE carrying which field failed, plus the
// underlying sentinel (or a %w-wrapped, more detailed version of one).
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Err)
}

// Unwrap is what makes errors.Is/errors.As able to see THROUGH FieldError
// to the sentinel underneath — without this method, FieldError would be a
// dead end in the wrap chain.
func (e *FieldError) Unwrap() error {
	return e.Err
}

// --- Validators -----------------------------------------------------

func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &FieldError{Field: field, Err: ErrRequired}
	}
	return nil
}

func MinLength(field, value string, min int) error {
	if len(value) < min {
		return &FieldError{Field: field, Err: fmt.Errorf("%w: need at least %d characters, got %d", ErrTooShort, min, len(value))}
	}
	return nil
}

func MaxLength(field, value string, max int) error {
	if len(value) > max {
		return &FieldError{Field: field, Err: fmt.Errorf("%w: max %d characters, got %d", ErrTooLong, max, len(value))}
	}
	return nil
}

func Email(field, value string) error {
	if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
		return &FieldError{Field: field, Err: ErrInvalidEmail}
	}
	return nil
}

func Range(field string, value, min, max int) error {
	if value < min || value > max {
		return &FieldError{Field: field, Err: fmt.Errorf("%w: must be between %d and %d, got %d", ErrOutOfRange, min, max, value)}
	}
	return nil
}

// All aggregates any number of validator results into ONE error using
// errors.Join (Go 1.20+) — nil results are skipped, and if everything
// passed, All itself returns nil. Crucially, errors.Join's result still
// supports errors.Is/errors.As drilling into EACH joined error, so nothing
// about wrapping is lost just because multiple failures got merged.
func All(checks ...error) error {
	var errs []error
	for _, c := range checks {
		if c != nil {
			errs = append(errs, c)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// Split flattens a joined error tree back into a []error — useful for
// callers that want to loop over every individual failure (e.g. to build a
// per-field error map for an API response) instead of checking for one
// specific kind with errors.Is/errors.As.
func Split(err error) []error {
	if err == nil {
		return nil
	}
	// errors.Join's result implements `Unwrap() []error` (plural) — a
	// different shape than the single-error `Unwrap() error` FieldError
	// implements above. This type assertion checks for that specific shape.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var out []error
		for _, sub := range joined.Unwrap() {
			out = append(out, Split(sub)...) // recurse, in case of nested joins
		}
		return out
	}
	return []error{err}
}
