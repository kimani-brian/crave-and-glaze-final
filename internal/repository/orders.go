package repository

import (
	"context"
	"crave-and-glaze/internal/models"
	"database/sql"
	"time"
)

type OrderModel struct {
	DB *sql.DB
}

// Create places a new order and its items into the database transactionally
func (m *OrderModel) Create(order *models.Order, items []models.OrderItem) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Updated SQL Insert
	stmt := `
		INSERT INTO orders (first_name, last_name, email, whatsapp_number, customer_phone, total_amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'PENDING', $7)
		RETURNING id
	`

	var newID int
	err = tx.QueryRowContext(ctx, stmt,
		order.FirstName,
		order.LastName,
		order.Email,
		order.WhatsappNumber,
		order.CustomerPhone, // MPESA Number
		order.TotalAmount,
		time.Now(),
	).Scan(&newID)

	if err != nil {
		return 0, err
	}

	// ... (The rest of the item insertion logic stays the same) ...

	stmtItem := `INSERT INTO order_items (order_id, product_variant_id, quantity, icing_flavor, custom_message, price_at_purchase) VALUES ($1, $2, $3, $4, $5, $6)`
	for _, item := range items {
		_, err = tx.ExecContext(ctx, stmtItem, newID, item.ProductVariantID, item.Quantity, item.IcingFlavor, item.CustomMessage, item.PriceAtPurchase)
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return newID, nil
}

// GetAll fetches all orders descending
func (m *OrderModel) GetAll() ([]models.Order, error) {
	stmt := `SELECT id, first_name, last_name, email, whatsapp_number, customer_phone, total_amount, status, created_at FROM orders ORDER BY id DESC`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		err = rows.Scan(&o.ID, &o.FirstName, &o.LastName, &o.Email, &o.WhatsappNumber, &o.CustomerPhone, &o.TotalAmount, &o.Status, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// UpdateStatus changes the order status
func (m *OrderModel) UpdateStatus(id int, status string) error {
	stmt := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := m.DB.Exec(stmt, status, id)
	return err
}

// Get Fetch a single order by ID
func (m *OrderModel) Get(id int) (*models.Order, error) {
	stmt := `SELECT id, first_name, last_name, email, whatsapp_number, customer_phone, total_amount, status, created_at FROM orders WHERE id = $1`
	o := &models.Order{}
	err := m.DB.QueryRow(stmt, id).Scan(&o.ID, &o.FirstName, &o.LastName, &o.Email, &o.WhatsappNumber, &o.CustomerPhone, &o.TotalAmount, &o.Status, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// OrderDetailItem helps us display the cake info nicely
type OrderDetailItem struct {
	ProductName string
	ImageURL    string
	WeightLabel string
	Quantity    int
	Price       float64
	Icing       string
	Message     string
}

// GetOrderItems fetches the cakes inside a specific order with their names
func (m *OrderModel) GetOrderItems(orderID int) ([]OrderDetailItem, error) {
	// Added p.image_url to the SELECT
	stmt := `
		SELECT 
			p.name, 
            p.image_url,
			pv.weight_label, 
			oi.quantity, 
			oi.price_at_purchase, 
			oi.icing_flavor, 
			oi.custom_message
		FROM order_items oi
		JOIN product_variants pv ON oi.product_variant_id = pv.id
		JOIN products p ON pv.product_id = p.id
		WHERE oi.order_id = $1
	`
	rows, err := m.DB.Query(stmt, orderID)
	// ... error handling ...
	defer rows.Close()

	var items []OrderDetailItem
	for rows.Next() {
		var i OrderDetailItem
		// Added &i.ImageURL to the Scan
		err = rows.Scan(&i.ProductName, &i.ImageURL, &i.WeightLabel, &i.Quantity, &i.Price, &i.Icing, &i.Message)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}
