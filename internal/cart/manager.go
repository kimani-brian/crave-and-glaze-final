package cart

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

// Item represents one line in the shopping cart
type Item struct {
	VariantID   int
	ProductName string
	ImageURL    string // We store name here to save DB lookups on simple views
	Price       float64
	Quantity    int
	Message     string
	Icing       string
}

// Get reads the cart from the cookie
func Get(r *http.Request) []Item {
	cookie, err := r.Cookie("crave_cart")
	if err != nil {
		return []Item{} // Return empty cart if no cookie exists
	}

	// Decode Base64
	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return []Item{}
	}

	// Unmarshal JSON
	var items []Item
	_ = json.Unmarshal(data, &items)
	return items
}

// Add appends a new item to the cart and updates the cookie
func Add(w http.ResponseWriter, r *http.Request, newItem Item) {
	items := Get(r)

	// Check if item already exists (same variant), if so, just add quantity
	found := false
	for i, item := range items {
		if item.VariantID == newItem.VariantID {
			items[i].Quantity += newItem.Quantity
			found = true
			break
		}
	}

	if !found {
		items = append(items, newItem)
	}

	saveCart(w, items)
}

// SaveCart writes the list back to the browser
func saveCart(w http.ResponseWriter, items []Item) {
	data, _ := json.Marshal(items)
	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     "crave_cart",
		Value:    encoded,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour), // Cart lasts 24 hours
		HttpOnly: true,                           // Javascript cannot access this (Security)
	})
}

// Total calculates the total cost
func Total(items []Item) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

// Remove deletes an item by its VariantID
func Remove(w http.ResponseWriter, r *http.Request, variantID int) {
	items := Get(r)
	var newItems []Item

	// Keep everything EXCEPT the one matching variantID
	for _, item := range items {
		if item.VariantID != variantID {
			newItems = append(newItems, item)
		}
	}

	saveCart(w, newItems)
}

// UpdateQuantity changes the quantity of a specific item
func UpdateQuantity(w http.ResponseWriter, r *http.Request, variantID int, change int) {
	items := Get(r)

	for i, item := range items {
		if item.VariantID == variantID {
			newQty := item.Quantity + change

			// Ensure quantity doesn't go below 1
			if newQty < 1 {
				newQty = 1
			}

			items[i].Quantity = newQty
			break
		}
	}

	saveCart(w, items)
}

// RemoveItem completely deletes an item (You might already have something like this)
func RemoveItem(w http.ResponseWriter, r *http.Request, variantID int) {
	items := Get(r)
	var newItems []Item

	for _, item := range items {
		if item.VariantID != variantID {
			newItems = append(newItems, item)
		}
	}

	saveCart(w, newItems)
}
