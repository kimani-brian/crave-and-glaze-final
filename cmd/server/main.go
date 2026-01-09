package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"crave-and-glaze/internal/cart"
	"crave-and-glaze/internal/daraja"
	"crave-and-glaze/internal/database"
	"crave-and-glaze/internal/models"
	"crave-and-glaze/internal/repository"
	"strconv"
)

// Application struct holds the dependencies for our app
type Application struct {
	Products *repository.ProductModel
	Orders   *repository.OrderModel
	Mpesa    *daraja.Service
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// 1. Init DB
	database.InitDB()

	mpesaService := daraja.NewService(
		os.Getenv("MPESA_KEY"),
		os.Getenv("MPESA_SECRET"),
	)

	// 2. Initialize Models/Repositories
	app := &Application{
		Products: &repository.ProductModel{DB: database.DB},
		Orders:   &repository.OrderModel{DB: database.DB},
		Mpesa:    mpesaService,
	}

	// 3. Setup Router
	mux := http.NewServeMux()

	// 3. Serve Static Files (Updated)
	// Get the current working directory to ensure we are looking in the right place
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web", "static"))

	// Print the path to the terminal so we can verify it's correct
	fmt.Println("Serving static files from:", filepath.Join(workDir, "web", "static"))

	fileServer := http.FileServer(filesDir)
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	// Pass 'app' methods to handlers
	mux.HandleFunc("/", app.homeHandler)

	mux.HandleFunc("/product", app.productHandler)

	mux.HandleFunc("POST /cart/add", app.addToCartHandler)
	mux.HandleFunc("GET /cart", app.viewCartHandler)

	mux.HandleFunc("GET /checkout", app.checkoutPageHandler)
	mux.HandleFunc("POST /checkout", app.placeOrderHandler)
	//paymentHandler
	mux.HandleFunc("GET /payment", app.paymentHandler)

	// Admin Routes
	mux.HandleFunc("GET /admin/dashboard", app.adminDashboardHandler)
	mux.HandleFunc("POST /admin/order/status", app.adminUpdateStatusHandler)

	mux.HandleFunc("GET /admin/products/add", app.adminAddProductPageHandler)
	mux.HandleFunc("POST /admin/products/add", app.adminAddProductHandler)

	mux.HandleFunc("GET /category", app.categoryHandler)

	// Admin Category Routes
	mux.HandleFunc("GET /admin/categories", app.adminCategoriesHandler)
	mux.HandleFunc("POST /admin/categories/add", app.adminAddCategoryHandler)
	mux.HandleFunc("POST /admin/categories/delete", app.adminDeleteCategoryHandler)

	mux.HandleFunc("GET /cakes", app.allCakesHandler)
	mux.HandleFunc("GET /admin/orders/view", app.adminOrderViewHandler)

	//cart remove
	mux.HandleFunc("POST /cart/remove", app.removeFromCartHandler)

	// 4. Start Server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("Crave & Glaze Server starting on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}

func (app *Application) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	products, err := app.Products.All()
	if err != nil {
		log.Println(err)
	}

	// Use the new TemplateData struct
	data := &models.TemplateData{
		Title:    "Home",
		Products: products,
	}

	// Use the new render helper
	app.render(w, r, "home.page.html", data)
}

func (app *Application) productHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	p, err := app.Products.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	variants, _ := app.Products.GetVariants(id)

	data := &models.TemplateData{
		Title:    p.Name,
		Product:  p,
		Variants: variants,
	}

	app.render(w, r, "product.page.html", data)
}
func (app *Application) addToCartHandler(w http.ResponseWriter, r *http.Request) {
	// Parse Form Data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	// Get IDs and convert to int
	productID, _ := strconv.Atoi(r.FormValue("product_id"))
	variantID, _ := strconv.Atoi(r.FormValue("variant_id"))
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))

	// Get Text inputs
	msg := r.FormValue("message")
	icing := r.FormValue("icing")

	// Validation
	if variantID == 0 || quantity < 1 {
		http.Error(w, "Please select a size", 400)
		return
	}

	// Fetch the specific variant to get the correct Price
	// (We need a helper for this in models, but for now let's reuse GetVariants
	// and loop, or write a quick GetVariantByID. For simplicity, let's trust the form
	// or fetch properly. Let's do it properly.)

	// TODO: Ideally, fetch the specific price from DB here to prevent tampering.
	// For this step, we will assume the price calculation happens at checkout
	// or fetch it now. Let's fetch the Product Name for display.
	product, _ := app.Products.Get(productID)

	// Quick hack: We need the price of the selected variant.
	// In a real app, create a method: app.Products.GetVariant(variantID)
	// Here, we fetch all and find the match.
	variants, _ := app.Products.GetVariants(productID)
	var selectedPrice float64
	var sizeLabel string

	for _, v := range variants {
		if v.ID == variantID {
			selectedPrice = v.Price
			sizeLabel = v.WeightLabel
			break
		}
	}

	// Create Item
	item := cart.Item{
		VariantID:   variantID,
		ProductName: product.Name + " (" + sizeLabel + ")",
		Price:       selectedPrice,
		Quantity:    quantity,
		Message:     msg,
		Icing:       icing,
	}

	// Save to Cookie
	cart.Add(w, r, item)

	// Redirect to Cart Page
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (app *Application) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get Cart Data
	items := cart.Get(r)
	total := cart.Total(items)

	// 2. Prepare the data wrapper
	// We create a temporary struct to hold both items and total
	cartData := struct {
		Items []cart.Item
		Total float64
	}{
		Items: items,
		Total: total,
	}

	// 3. Create the Template Data
	data := &models.TemplateData{
		Title: "Your Cart",
		Data:  cartData, // Pass the cart info into the generic 'Data' field
	}

	// 4. Render using the helper
	app.render(w, r, "cart.page.html", data)
}

func (app *Application) checkoutPageHandler(w http.ResponseWriter, r *http.Request) {
	items := cart.Get(r)
	if len(items) == 0 {
		http.Redirect(w, r, "/cakes", http.StatusSeeOther)
		return
	}

	total := cart.Total(items)

	// Prepare data just like the cart
	checkoutData := struct {
		Items []cart.Item
		Total float64
	}{
		Items: items,
		Total: total,
	}

	data := &models.TemplateData{
		Title: "Checkout",
		Data:  checkoutData,
	}

	app.render(w, r, "checkout.page.html", data)
}

func (app *Application) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	// 1. Get Customer Info
	name := r.FormValue("name")
	phone := r.FormValue("phone")

	// Basic Validation
	if name == "" || phone == "" {
		http.Error(w, "Name and Phone are required", 400)
		return
	}

	// 2. Get Cart Data
	cartItems := cart.Get(r)
	if len(cartItems) == 0 {
		http.Redirect(w, r, "/cakes", http.StatusSeeOther)
		return
	}
	total := cart.Total(cartItems)

	// 3. Prepare Order Model
	order := &models.Order{
		CustomerName:  name,
		CustomerPhone: phone,
		TotalAmount:   total,
	}

	// 4. Convert Cart Items to Order Items
	var orderItems []models.OrderItem
	for _, ci := range cartItems {
		orderItems = append(orderItems, models.OrderItem{
			ProductVariantID: ci.VariantID,
			Quantity:         ci.Quantity,
			IcingFlavor:      ci.Icing,
			CustomMessage:    ci.Message,
			PriceAtPurchase:  ci.Price,
		})
	}

	// 5. Save to Database
	orderID, err := app.Orders.Create(order, orderItems)
	if err != nil {
		log.Println("Failed to create order:", err)
		http.Error(w, "Failed to place order", 500)
		return
	}

	// 6. Clear the Cart (Expire the cookie)
	http.SetCookie(w, &http.Cookie{
		Name:   "crave_cart",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// 7. Redirect to Payment Page (Passing the Order ID)
	// We use Sprintf to put the ID in the URL
	redirectURL := fmt.Sprintf("/payment?order_id=%d", orderID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *Application) paymentHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get Order ID
	orderIDStr := r.URL.Query().Get("order_id")
	var orderID int
	fmt.Sscanf(orderIDStr, "%d", &orderID)

	// 2. Fetch Order Details
	var phone string
	var amount float64
	var status string

	// Safety check: ensure we actually got an ID
	if orderID == 0 {
		http.Error(w, "Invalid Order ID", 400)
		return
	}

	err := app.Orders.DB.QueryRow("SELECT customer_phone, total_amount, status FROM orders WHERE id=$1", orderID).Scan(&phone, &amount, &status)
	if err != nil {
		http.Error(w, "Order not found in Database", 404)
		return
	}

	// 3. Initiate STK Push (Only if pending)
	if status == "PENDING" {
		// Format phone to 254...
		if len(phone) > 0 && phone[0] == '0' {
			phone = "254" + phone[1:]
		}

		// Trigger MPESA
		_ = app.Mpesa.InitiateSTKPush(phone, amount, orderID)
	}

	// 4. Prepare Data
	// We use a manual struct here because this page is unique
	data := struct {
		OrderID    int
		Phone      string
		Amount     float64
		Categories []models.Category // Add this so the Navbar works!
	}{
		OrderID: orderID,
		Phone:   phone,
		Amount:  amount,
	}

	// Fetch categories for the navbar so it's not empty
	data.Categories, _ = app.Products.GetAllCategories()

	// 5. Render Templates (With Error Checking!)
	files := []string{
		"./web/templates/base.layout.html",
		"./web/templates/payment.page.html",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		// This will print the EXACT error to your terminal
		log.Println("Error loading payment page:", err)
		http.Error(w, "Error loading template: "+err.Error(), 500)
		return
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Println("Error executing template:", err)
	}
}

func (app *Application) adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Security: In a real app, check for a session cookie here!
	// For now, we assume anyone accessing this URL is the admin (Localhost dev mode).

	orders, err := app.Orders.GetAll()
	if err != nil {
		log.Println(err)
		http.Error(w, "Server Error", 500)
		return
	}

	data := struct {
		Orders []models.Order
	}{
		Orders: orders,
	}

	files := []string{
		"./web/templates/admin/admin.layout.html",
		"./web/templates/admin/dashboard.page.html",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println(err)
		http.Error(w, "Template Error", 500)
		return
	}

	ts.ExecuteTemplate(w, "admin_base", data)
}

func (app *Application) adminUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("order_id")
	status := r.FormValue("status")

	var id int
	fmt.Sscanf(idStr, "%d", &id)

	err := app.Orders.UpdateStatus(id, status)
	if err != nil {
		log.Println("Error updating status:", err)
	}

	// Redirect back to dashboard
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// Add this helper function to your Application struct or as a standalone
// render is our centralized HTML generator
func (app *Application) render(w http.ResponseWriter, r *http.Request, page string, data *models.TemplateData) {
	// 1. Fetch Categories for the Navbar (Every page needs this)
	cats, err := app.Products.GetAllCategories()
	if err != nil {
		log.Println("Error fetching categories:", err)
	}
	data.Categories = cats

	// 2. Set Default Data
	data.CurrentYear = time.Now().Year()

	// 3. Parse Templates
	// We combine the base layout with the specific page requested
	files := []string{
		"./web/templates/base.layout.html",
		"./web/templates/" + page,
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println("Template Parse Error:", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// 4. Execute
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Println("Template Execute Error:", err)
	}
}

func (app *Application) adminAddProductPageHandler(w http.ResponseWriter, r *http.Request) {
	// We need categories for the dropdown
	cats, _ := app.Products.GetAllCategories()
	app.render(w, r, "admin/add_product.page.html", &models.TemplateData{
		Title:      "Add New Cake",
		Categories: cats,
	})
}

func (app *Application) adminAddProductHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form (Max 10MB)
	r.ParseMultipartForm(10 << 20)

	// 2. Handle Image Upload
	file, handler, err := r.FormFile("image")
	var imagePath string

	if err == nil {
		defer file.Close()
		// Create unique name or keep original
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
		filePath := "./web/static/uploads/" + filename

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error saving file", 500)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)

		imagePath = "/static/uploads/" + filename
	} else {
		imagePath = "/static/img/cake-placeholder.jpg"
	}

	// 3. Get Basic Info
	name := r.FormValue("name")
	desc := r.FormValue("description")
	catID, _ := strconv.Atoi(r.FormValue("category_id"))

	// 4. Save Product
	p := models.Product{
		Name:        name,
		Description: desc,
		Category:    strconv.Itoa(catID), // Storing ID in the struct field temporarily
		ImageURL:    imagePath,
	}

	newID, err := app.Products.InsertProduct(p)
	if err != nil {
		log.Println("Insert Error:", err)
		http.Error(w, "Database Error", 500)
		return
	}

	// 5. Handle Sizes & Prices
	// We expect form fields like: weight_1, price_1, weight_2, price_2
	// A simple loop for 3 possible options (you can make this dynamic with JS later)
	for i := 1; i <= 3; i++ {
		weight := r.FormValue(fmt.Sprintf("weight_%d", i))
		priceStr := r.FormValue(fmt.Sprintf("price_%d", i))

		if weight != "" && priceStr != "" {
			price, _ := strconv.ParseFloat(priceStr, 64)
			app.Products.InsertVariant(newID, weight, price)
		}
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

func (app *Application) categoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	// Fetch products for this category
	products, err := app.Products.GetByCategory(id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Server Error", 500)
		return
	}

	data := &models.TemplateData{
		Title:    "Category",
		Products: products,
	}

	app.render(w, r, "category.page.html", data)
}
func (app *Application) adminCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	// We reuse the GetAllCategories method we made earlier
	cats, err := app.Products.GetAllCategories()
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}

	data := &models.TemplateData{
		Title:      "Manage Categories",
		Categories: cats, // This fills the table
		IsAdmin:    true, // Logic to show admin sidebar/menu
	}

	app.render(w, r, "admin/categories.page.html", data)
}

func (app *Application) adminAddCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	name := r.FormValue("name")
	// Generate a simple slug (e.g., "Wedding Cakes" -> "wedding-cakes")
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	err := app.Products.InsertCategory(name, slug)
	if err != nil {
		log.Println("Error adding category:", err)
	}

	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

func (app *Application) adminDeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, _ := strconv.Atoi(idStr)

	err := app.Products.DeleteCategory(id)
	if err != nil {
		log.Println("Error deleting category:", err)
	}

	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}
func (app *Application) allCakesHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch All Products (We created this method in Phase 3)
	products, err := app.Products.All()
	if err != nil {
		log.Println("Error fetching all cakes:", err)
		http.Error(w, "Server Error", 500)
		return
	}

	// 2. Prepare Data
	data := &models.TemplateData{
		Title:    "All Cakes",
		Products: products,
	}

	// 3. Render (Reusing the category template because it's just a grid of cakes)
	app.render(w, r, "category.page.html", data)
}

func (app *Application) adminOrderViewHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	// 1. Get the Order Basic Info
	order, err := app.Orders.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// 2. Get the Items (Cakes) in that order
	items, err := app.Orders.GetOrderItems(id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Server Error", 500)
		return
	}

	data := &models.TemplateData{
		Title:      "Order Details",
		Order:      order,
		OrderItems: items,
		IsAdmin:    true,
	}

	app.render(w, r, "admin/order_details.page.html", data)
}

func (app *Application) removeFromCartHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the Variant ID from the form
	idStr := r.FormValue("variant_id")
	id, _ := strconv.Atoi(idStr)

	// Remove it
	cart.Remove(w, r, id)

	// Refresh the page
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}
