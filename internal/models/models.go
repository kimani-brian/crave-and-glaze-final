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
	Value interface{} `json:"Value"`
}
