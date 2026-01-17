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
	ID             int
	FirstName      string // Was CustomerName
	LastName       string // New
	Email          string // New
	WhatsappNumber string // New
	CustomerPhone  string // This remains the MPESA Payment Number
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

// TemplateData holds data sent from Go to HTML

type TemplateData struct {
	Title       string
	CurrentYear int
	Categories  []Category
	Products    []Product
	Product     *Product
	Variants    []ProductVariant
	Items       interface{} // Generic field to hold Cart Items
	Total       float64     // Total Price
	Order       *Order
	OrderItems  interface{}
	IsAdmin     bool
}

// Category struct
type Category struct {
	ID   int
	Name string
	Slug string
}

// --- MPESA Callback Structures ---

type MpesaCallbackResponse struct {
	Body struct {
		StkCallback StkCallback `json:"stkCallback"`
	} `json:"Body"`
}

type StkCallback struct {
	MerchantRequestID string           `json:"MerchantRequestID"`
	CheckoutRequestID string           `json:"CheckoutRequestID"`
	ResultCode        int              `json:"ResultCode"`
	ResultDesc        string           `json:"ResultDesc"`
	CallbackMetadata  CallbackMetadata `json:"CallbackMetadata"`
}

type CallbackMetadata struct {
	Item []CallbackItem `json:"Item"`
}

type CallbackItem struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value"` // interface{} because Value can be string or float
}
