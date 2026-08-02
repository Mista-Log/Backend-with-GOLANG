// Exercise 3: Unit Converter
//
// Converts between units of length, weight, and volume.
// Usage: go run main.go -val 5 -from km -to mi
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// unitDef stores how many "base units" one of this unit equals, plus which
// category (length/weight/volume) it belongs to — this is the same
// "table-driven design" idea used in the File Organizer's categoryFor map.
type unitDef struct {
	category string
	toBase   float64 // multiply by this to convert TO the base unit
}

// Base units chosen per category: meters (length), grams (weight),
// milliliters (volume). Every unit's factor says "1 of me = how many base units".
var units = map[string]unitDef{
	// length — base: meters
	"m":  {"length", 1},
	"km": {"length", 1000},
	"cm": {"length", 0.01},
	"mi": {"length", 1609.34},
	"ft": {"length", 0.3048},
	"in": {"length", 0.0254},

	// weight — base: grams
	"g":  {"weight", 1},
	"kg": {"weight", 1000},
	"lb": {"weight", 453.592},
	"oz": {"weight", 28.3495},

	// volume — base: milliliters
	"ml": {"volume", 1},
	"l":  {"volume", 1000},
	"gal": {"volume", 3785.41},
	"cup": {"volume", 236.588},
}

// convert looks up both units, checks they're the same category (converting
// kilograms to miles should be a clear error, not a silent wrong answer),
// then converts value -> base unit -> target unit.
func convert(value float64, from, to string) (float64, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	fromDef, ok := units[from]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q", from)
	}
	toDef, ok := units[to]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q", to)
	}
	if fromDef.category != toDef.category {
		return 0, fmt.Errorf("cannot convert %s (%s) to %s (%s) — different categories",
			from, fromDef.category, to, toDef.category)
	}

	baseValue := value * fromDef.toBase
	return baseValue / toDef.toBase, nil
}

func main() {
	val := flag.Float64("val", 0, "value to convert")
	from := flag.String("from", "", "source unit (m, km, cm, mi, ft, in, g, kg, lb, oz, ml, l, gal, cup)")
	to := flag.String("to", "", "target unit")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "Error: -from and -to are required")
		flag.Usage()
		os.Exit(1)
	}

	result, err := convert(*val, *from, *to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Printf("%.4g %s = %.4g %s\n", *val, *from, result, *to)
}
