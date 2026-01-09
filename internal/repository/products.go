package repository

import (
	"crave-and-glaze/internal/models"
	"database/sql"
	"log"
)

type ProductModel struct {
	DB *sql.DB
}

// All returns all active products with their starting price (lowest variant price)
func (m *ProductModel) All() ([]models.Product, error) {
	// This query joins products and variants to find the cheapest option for each cake
	// COALESCE(MIN(v.price), 0) ensures we don't crash if a product has no variants yet
	stmt := `
		SELECT p.id, p.name, p.description, p.image_url, c.name as category, COALESCE(MIN(v.price), 0) as starting_price
		FROM products p
		LEFT JOIN product_variants v ON p.id = v.product_id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.is_active = true
		GROUP BY p.id, p.name, p.description, p.image_url, c.name
		ORDER BY p.id DESC
	`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product

	for rows.Next() {
		var p models.Product
		// We use sql.NullString in case description/image is NULL in DB,
		// but for simplicity here we assume they are filled or handle errors.
		err = rows.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.Category, &p.StartingPrice)
		if err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		products = append(products, p)
	}

	return products, nil
}

// Get fetches a single product by ID
func (m *ProductModel) Get(id int) (*models.Product, error) {
	stmt := `
		SELECT id, name, description, image_url, category_id 
		FROM products 
		WHERE id = $1 AND is_active = true
	`
	row := m.DB.QueryRow(stmt, id)

	p := &models.Product{}
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.Category) // Category here is just the ID int for now or we ignore it
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetVariants fetches all size options for a specific product
func (m *ProductModel) GetVariants(productID int) ([]models.ProductVariant, error) {
	stmt := `SELECT id, weight_label, price FROM product_variants WHERE product_id = $1 ORDER BY price ASC`

	rows, err := m.DB.Query(stmt, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []models.ProductVariant
	for rows.Next() {
		var v models.ProductVariant
		err = rows.Scan(&v.ID, &v.WeightLabel, &v.Price)
		if err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	return variants, nil
}

// GetAllCategories fetches all categories for the navbar
func (m *ProductModel) GetAllCategories() ([]models.Category, error) {
	stmt := `SELECT id, name, slug FROM categories ORDER BY name ASC`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		err = rows.Scan(&c.ID, &c.Name, &c.Slug)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetByCategory fetches active products for a specific category
func (m *ProductModel) GetByCategory(categoryID int) ([]models.Product, error) {
	stmt := `
		SELECT p.id, p.name, p.description, p.image_url, c.name, COALESCE(MIN(v.price), 0)
		FROM products p
		LEFT JOIN product_variants v ON p.id = v.product_id
		JOIN categories c ON p.category_id = c.id
		WHERE p.category_id = $1 AND p.is_active = true
		GROUP BY p.id, p.name, p.description, p.image_url, c.name
		ORDER BY p.id DESC
	`
	rows, err := m.DB.Query(stmt, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		err = rows.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.Category, &p.StartingPrice)
		if err != nil {
			continue // Skip bad rows
		}
		products = append(products, p)
	}
	return products, nil
}

// InsertProduct saves the main cake info and returns the new ID
func (m *ProductModel) InsertProduct(p models.Product) (int, error) {
	// Note: We use the 'category_id' column, so we pass the ID, not the name
	stmt := `
		INSERT INTO products (name, description, category_id, image_url, is_active) 
		VALUES ($1, $2, $3, $4, true) 
		RETURNING id
	`
	var newID int
	// p.Category here holds the Category ID as a string from the form
	err := m.DB.QueryRow(stmt, p.Name, p.Description, p.Category, p.ImageURL).Scan(&newID)
	return newID, err
}

// InsertVariant saves a specific size and price (e.g., 1KG - 4000)
func (m *ProductModel) InsertVariant(productID int, label string, price float64) error {
	stmt := `INSERT INTO product_variants (product_id, weight_label, price) VALUES ($1, $2, $3)`
	_, err := m.DB.Exec(stmt, productID, label, price)
	return err
}

// InsertCategory adds a new category
func (m *ProductModel) InsertCategory(name, slug string) error {
	stmt := `INSERT INTO categories (name, slug) VALUES ($1, $2)`
	_, err := m.DB.Exec(stmt, name, slug)
	return err
}

// DeleteCategory removes a category
func (m *ProductModel) DeleteCategory(id int) error {
	stmt := `DELETE FROM categories WHERE id = $1`
	_, err := m.DB.Exec(stmt, id)
	return err
}
