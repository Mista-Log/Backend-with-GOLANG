CREATE TABLE products (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	sku      TEXT NOT NULL UNIQUE,
	name     TEXT NOT NULL,
	category TEXT NOT NULL,
	price    REAL NOT NULL CHECK (price > 0),
	quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0)
);

-- Products are looked up/filtered by category constantly (see
-- ListByCategory) — this index is what keeps that fast as the catalog grows.
CREATE INDEX idx_products_category ON products(category);
