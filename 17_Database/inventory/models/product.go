package models

// db tags tell sqlx which column maps to which field — the guide's `db:"..."`
// convention, directly analogous to `json:"..."` from Module 11.
type Product struct {
	ID       int     `db:"id"`
	SKU      string  `db:"sku"`
	Name     string  `db:"name"`
	Category string  `db:"category"`
	Price    float64 `db:"price"`
	Quantity int     `db:"quantity"`
}
