package models

type Product struct {
	ID            int
	Name          string
	Description   string
	ImageURL      string
	Category      string
	StartingPrice float64
}

type ProductVariant struct {
	ID          int
	ProductID   int
	WeightLabel string
	Price       float64
}
type Order struct {
	ID             int
	FirstName      string
	LastName       string
	Email          string
	WhatsappNumber string
	CustomerPhone  string
	TotalAmount    float64
	Status         string
	MpesaReceipt   string
	CreatedAt      string
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

type TemplateData struct {
	Title       string
	CurrentYear int
	Categories  []Category
	Products    []Product
	Product     *Product
	Variants    []ProductVariant
	Items       interface{}
	Total       float64
	Order       *Order
	OrderItems  interface{}
	IsAdmin     bool
	CartCount   int
	WhatsAppURL string
	// Review-step contact details (POST /checkout renders, POST /checkout/confirm consumes)
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// Category struct
type Category struct {
	ID   int
	Name string
	Slug string
}
