package bank

import "testing"

// FuzzParseAmount hunts for inputs that make ParseAmount panic, or that
// parse "successfully" into something whose own String() method then
// misbehaves — exactly the pattern from Module 14's guide, section 4.
func FuzzParseAmount(f *testing.F) {
	// Seed corpus: a few known-interesting starting points for the fuzzer
	// to mutate from.
	f.Add("19.99")
	f.Add("$19.99")
	f.Add("0")
	f.Add("-5.00")
	f.Add("")
	f.Add("$")
	f.Add("not-a-number")
	f.Add("999999999999999999999999.99") // deliberately huge

	f.Fuzz(func(t *testing.T, input string) {
		amount, err := ParseAmount(input)
		if err != nil {
			return // an error is a perfectly valid outcome for bad input
		}

		// Whatever DID successfully parse should always produce a
		// non-empty, well-formed String() — if this ever fails, the
		// fuzzer found an input that "succeeds" but produces garbage.
		s := amount.String()
		if s == "" {
			t.Errorf("ParseAmount(%q) succeeded but String() was empty", input)
		}
		if s[0] != '$' {
			t.Errorf("ParseAmount(%q) -> String() = %q; expected it to start with $", input, s)
		}
	})
}
