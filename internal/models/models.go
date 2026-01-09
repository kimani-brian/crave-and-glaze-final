package models

// Product represents the general cake details
type Product struct {
	ID            int
	Name          string
	Description   string
	ImageURL      string
	Category      string  // We might fetch the category name via JOIN
	StartingPrice float64 // Calculated field (min price of variants)
}

// ProductVariant represents the specific size/price options (e.g., 1KG = 4000)
type ProductVariant struct {
	ID          int
	ProductID   int
	WeightLabel string // e.g., "1 Kg"
	Price       float64
}
type Order struct {
	ID            int
	CustomerName  string
	CustomerPhone string
	TotalAmount   float64
	Status        string // PENDING, PAID, FAILED
	MpesaReceipt  string
	CreatedAt     string // or time.Time
}

type OrderItem struct {
	ID               int
	OrderID          int
	ProductVariantID int
	Quantity         int
	IcingFlavor      string
	CustomMessage    string
	PriceAtPurchase  float64
}

// TemplateData holds data sent from Go to HTML
type TemplateData struct {
	Title       string
	CurrentYear int
	Categories  []Category       // For the Navbar dropdown
	Products    []Product        // For lists of cakes
	Product     *Product         // For single cake details
	Variants    []ProductVariant // For single cake size options
	Orders      []Order          // For Admin Dashboard
	Order       *Order           // For Admin Order Details
	IsAdmin     bool             // To show/hide Admin links
	CartTotal   float64          // To show cart total in navbar
	CartCount   int
	OrderItems  interface{} // To show number of items
	Data        interface{}
}

// Category struct
type Category struct {
	ID   int
	Name string
	Slug string
}
