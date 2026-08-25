package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"inventory/models"
)

// ProductRepository is the interface every consumer of this package
// depends on — Module 06's interfaces, applied to the data layer (the
// guide's Repository Pattern section). Business logic in main.go never
// imports sqlx or knows a database is involved at all.
type ProductRepository interface {
	Create(ctx context.Context, p *models.Product) (int, error)
	GetByID(ctx context.Context, id int) (*models.Product, error)
	GetBySKU(ctx context.Context, sku string) (*models.Product, error)
	ListByCategory(ctx context.Context, category string) ([]models.Product, error)
	AdjustQuantity(ctx context.Context, id int, delta int) error
	Delete(ctx context.Context, id int) error
}

// SQLiteProductRepository is the REAL implementation, backed by sqlx.
type SQLiteProductRepository struct {
	db *sqlx.DB
}

func NewSQLiteProductRepository(db *sqlx.DB) *SQLiteProductRepository {
	return &SQLiteProductRepository{db: db}
}

func (r *SQLiteProductRepository) Create(ctx context.Context, p *models.Product) (int, error) {
	// NamedExecContext binds directly from the struct's `db` tags —
	// no manual argument-by-position list to keep in sync with the SQL.
	result, err := r.db.NamedExecContext(ctx,
		`INSERT INTO products (sku, name, category, price, quantity)
		 VALUES (:sku, :name, :category, :price, :quantity)`, p)
	if err != nil {
		return 0, fmt.Errorf("creating product: %w", err)
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func (r *SQLiteProductRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	var p models.Product
	// GetContext scans directly into the struct via `db` tags — no
	// Scan(&p.ID, &p.SKU, &p.Name, ...) boilerplate at all.
	err := r.db.GetContext(ctx, &p, "SELECT * FROM products WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no product with id %d", id)
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLiteProductRepository) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	var p models.Product
	err := r.db.GetContext(ctx, &p, "SELECT * FROM products WHERE sku = ?", sku)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no product with sku %q", sku)
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLiteProductRepository) ListByCategory(ctx context.Context, category string) ([]models.Product, error) {
	var products []models.Product
	// SelectContext scans MULTIPLE rows directly into a slice of structs —
	// the sqlx equivalent of Query + a manual rows.Next() loop.
	err := r.db.SelectContext(ctx, &products, "SELECT * FROM products WHERE category = ? ORDER BY name", category)
	return products, err
}

func (r *SQLiteProductRepository) AdjustQuantity(ctx context.Context, id int, delta int) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE products SET quantity = quantity + ? WHERE id = ? AND quantity + ? >= 0", delta, id, delta)
	if err != nil {
		return fmt.Errorf("adjusting quantity: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no product with id %d, or adjustment would make quantity negative", id)
	}
	return nil
}

func (r *SQLiteProductRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no product with id %d", id)
	}
	return nil
}
