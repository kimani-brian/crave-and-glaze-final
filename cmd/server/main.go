package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"crave-and-glaze/internal/cart"
	"crave-and-glaze/internal/database"
	"crave-and-glaze/internal/mailer"
	"crave-and-glaze/internal/models"
	"crave-and-glaze/internal/repository"
	"strconv"
)

// Application struct holds the dependencies for our app
type Application struct {
	Products *repository.ProductModel
	Orders   *repository.OrderModel
	Users    *repository.UserModel
	Mailer   *mailer.Mailer
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, assuming production environment.")
	}
	// 1. Init DB
	database.InitDB()

	mailService := mailer.New(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
	)

	// 2. Initialize Models/Repositories
	app := &Application{
		Products: &repository.ProductModel{DB: database.DB},
		Orders:   &repository.OrderModel{DB: database.DB},
		Users:    &repository.UserModel{DB: database.DB},
		Mailer:   mailService,
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

	// FIX: Added "GET " to the start of the pattern to prevent conflict with "GET /"
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// ==========================================
	// PUBLIC ROUTES (No Login Required)
	// ==========================================

	// Home & Products
	mux.HandleFunc("GET /", app.homeHandler)
	mux.HandleFunc("GET /cakes", app.allCakesHandler)
	mux.HandleFunc("GET /category", app.categoryHandler)
	mux.HandleFunc("GET /product", app.productHandler)

	// Cart Functions
	mux.HandleFunc("POST /cart/add", app.addToCartHandler)
	mux.HandleFunc("GET /cart", app.viewCartHandler)
	mux.HandleFunc("POST /cart/remove", app.removeFromCartHandler)
	mux.HandleFunc("POST /cart/update", app.updateCartHandler)

	// Checkout & Order Confirmation
	mux.HandleFunc("GET /checkout", app.checkoutPageHandler)
	mux.HandleFunc("POST /checkout", app.placeOrderHandler)
	mux.HandleFunc("POST /checkout/confirm", app.confirmOrderHandler)
	mux.HandleFunc("GET /order-confirmed", app.orderConfirmedHandler)

	// Authentication
	mux.HandleFunc("GET /admin/login", app.loginPageHandler)
	mux.HandleFunc("POST /admin/login", app.loginPostHandler)
	mux.HandleFunc("POST /admin/logout", app.logoutHandler)

	// ==========================================
	// ADMIN ROUTES (Protected by Middleware)
	// ==========================================

	// Dashboard & Orders
	mux.HandleFunc("GET /admin/dashboard", app.requireAdmin(app.adminDashboardHandler))
	mux.HandleFunc("POST /admin/order/status", app.requireAdmin(app.adminUpdateStatusHandler))
	mux.HandleFunc("GET /admin/orders/view", app.requireAdmin(app.adminOrderViewHandler))

	// Category Management
	mux.HandleFunc("GET /admin/categories", app.requireAdmin(app.adminCategoriesHandler))
	mux.HandleFunc("POST /admin/categories/add", app.requireAdmin(app.adminAddCategoryHandler))
	mux.HandleFunc("POST /admin/categories/delete", app.requireAdmin(app.adminDeleteCategoryHandler))

	// Product Management
	mux.HandleFunc("GET /admin/products", app.requireAdmin(app.adminProductsListHandler))
	mux.HandleFunc("GET /admin/products/add", app.requireAdmin(app.adminAddProductPageHandler))
	mux.HandleFunc("POST /admin/products/add", app.requireAdmin(app.adminAddProductHandler))
	mux.HandleFunc("GET /admin/products/edit", app.requireAdmin(app.adminEditProductPageHandler))
	mux.HandleFunc("POST /admin/products/edit", app.requireAdmin(app.adminEditProductHandler))
	mux.HandleFunc("POST /admin/products/delete", app.requireAdmin(app.adminDeleteProductHandler))
	//search route
	mux.HandleFunc("GET /search", app.searchHandler)
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
		ImageURL:    product.ImageURL,
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

	// 2. Create the Template Data
	// We assign directly to Items and Total, just like in the Checkout handler
	data := &models.TemplateData{
		Title: "Your Cart",
		Items: items,
		Total: total,
	}

	// 3. Render using the helper
	app.render(w, r, "cart.page.html", data)
}

func (app *Application) checkoutPageHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get Cart Items
	items := cart.Get(r)
	if len(items) == 0 {
		http.Redirect(w, r, "/cakes", http.StatusSeeOther)
		return
	}

	// 2. Calculate Total
	total := cart.Total(items)

	// 3. Prepare Data using the master TemplateData struct
	data := &models.TemplateData{
		Title: "Checkout",
		Items: items, // <--- Pass items here
		Total: total, // <--- Pass total here
	}

	// 4. Render using the helper (Fixes Navbar & Layout)
	app.render(w, r, "checkout.page.html", data)
}

func (app *Application) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	// 1. Get Customer Info
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	whatsapp := r.FormValue("whatsapp")

	if firstName == "" || whatsapp == "" {
		http.Error(w, "Name and WhatsApp number are required", 400)
		return
	}

	// 2. Get Cart
	cartItems := cart.Get(r)
	if len(cartItems) == 0 {
		http.Redirect(w, r, "/cakes", http.StatusSeeOther)
		return
	}
	total := cart.Total(cartItems)

	// 3. Review step — nothing is saved yet. The customer confirms
	// on the next page, which re-posts these details to /checkout/confirm.
	data := &models.TemplateData{
		Title:     "Confirm Your Order",
		Items:     cartItems,
		Total:     total,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     whatsapp,
	}
	app.render(w, r, "confirm.page.html", data)
}

func (app *Application) confirmOrderHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	// 1. Get Customer Info (re-posted as hidden fields from the review page)
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	whatsapp := r.FormValue("whatsapp")

	if firstName == "" || whatsapp == "" {
		http.Redirect(w, r, "/checkout", http.StatusSeeOther)
		return
	}

	// 2. Get Cart (still in the cookie — it is only cleared after placing)
	cartItems := cart.Get(r)
	if len(cartItems) == 0 {
		http.Redirect(w, r, "/cakes", http.StatusSeeOther)
		return
	}
	total := cart.Total(cartItems)

	// 3. Prepare Order Model
	order := &models.Order{
		FirstName:      firstName,
		LastName:       lastName,
		Email:          email,
		WhatsappNumber: whatsapp,
		CustomerPhone:  whatsapp,
		TotalAmount:    total,
	}

	// 4. Convert Items
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

	// 6. Send Emails (customer receipt + admin alert) in the background.
	app.sendOrderEmails(orderID, firstName, lastName, email, whatsapp, total)

	// 7. Clear the Cart
	http.SetCookie(w, &http.Cookie{
		Name:   "crave_cart",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// 8. Redirect to Confirmation (customer confirms the order via WhatsApp there)
	redirectURL := fmt.Sprintf("/order-confirmed?id=%d", orderID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// sendOrderEmails fires the customer receipt and admin alert without blocking.
func (app *Application) sendOrderEmails(orderID int, firstName, lastName, email, whatsapp string, total float64) {
	detailItems, _ := app.Orders.GetOrderItems(orderID)
	emailData := struct {
		ID            int
		CustomerName  string
		CustomerPhone string
		TotalAmount   float64
		Items         interface{}
	}{
		ID:            orderID,
		CustomerName:  firstName + " " + lastName,
		CustomerPhone: whatsapp,
		TotalAmount:   total,
		Items:         detailItems,
	}
	if email != "" {
		go app.Mailer.Send(email, "Order Received - Crave & Glaze #"+fmt.Sprint(orderID), "customer_receipt.html", emailData)
	}
	if adminEmail := os.Getenv("ADMIN_EMAIL"); adminEmail != "" {
		go app.Mailer.Send(adminEmail, "🔔 New Order #"+fmt.Sprint(orderID), "admin_alert.html", emailData)
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

// render is our centralized HTML generator
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

	// 3. CALCULATE CART COUNT
	// We get the items from the cookie using the 'cart' package
	currentCart := cart.Get(r)
	totalQty := 0
	for _, item := range currentCart {
		totalQty += item.Quantity
	}
	data.CartCount = totalQty

	// 4. Parse Templates
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

	// 5. Execute
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

func (app *Application) updateCartHandler(w http.ResponseWriter, r *http.Request) {
	// Parse IDs
	variantID, _ := strconv.Atoi(r.FormValue("variant_id"))
	action := r.FormValue("action") // will be "increase" or "decrease"

	switch action {
	case "increase":
		cart.UpdateQuantity(w, r, variantID, 1)
	case "decrease":
		cart.UpdateQuantity(w, r, variantID, -1)
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (app *Application) removeCartHandler(w http.ResponseWriter, r *http.Request) {
	variantID, _ := strconv.Atoi(r.FormValue("variant_id"))
	cart.RemoveItem(w, r, variantID)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// 1. List all products
func (app *Application) adminProductsListHandler(w http.ResponseWriter, r *http.Request) {
	products, err := app.Products.All() // We reuse the All() function
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}

	app.render(w, r, "admin/products.page.html", &models.TemplateData{
		Title:    "Manage Products",
		Products: products,
		IsAdmin:  true,
	})
}

// 2. Delete a product
func (app *Application) adminDeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	app.Products.DeleteProduct(id)
	http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}

// 3. Show Edit Page
func (app *Application) adminEditProductPageHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	// Get Product
	p, err := app.Products.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Get Variants (Prices)
	variants, _ := app.Products.GetVariants(id)

	// Get Categories (for dropdown)
	cats, _ := app.Products.GetAllCategories()

	app.render(w, r, "admin/edit_product.page.html", &models.TemplateData{
		Title:      "Edit Product",
		Product:    p,
		Variants:   variants,
		Categories: cats,
		IsAdmin:    true,
	})
}

// 4. Process the Update
func (app *Application) adminEditProductHandler(w http.ResponseWriter, r *http.Request) {
	// Parse Multipart Form (for image upload)
	r.ParseMultipartForm(10 << 20)

	id, _ := strconv.Atoi(r.FormValue("id"))
	name := r.FormValue("name")
	desc := r.FormValue("description")
	catID := r.FormValue("category_id")

	// Handle Image (Check if user uploaded a new one)
	file, handler, err := r.FormFile("image")
	var imagePath string

	if err == nil {
		defer file.Close()
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
		// Keep existing image if no new one uploaded
		imagePath = r.FormValue("existing_image")
	}

	// Update Main Product
	p := models.Product{
		ID:          id,
		Name:        name,
		Description: desc,
		Category:    catID,
		ImageURL:    imagePath,
	}
	app.Products.UpdateProduct(p)

	// Update Variant Prices (We loop through the form inputs)
	// We expect inputs named "price_VARIANT_ID"
	for k, v := range r.PostForm {
		if strings.HasPrefix(k, "price_") {
			// Extract variant ID from string "price_12"
			variantIDStr := strings.TrimPrefix(k, "price_")
			variantID, _ := strconv.Atoi(variantIDStr)
			newPrice, _ := strconv.ParseFloat(v[0], 64)

			app.Products.UpdateVariantPrice(variantID, newPrice)
		}
	}

	http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}

func (app *Application) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "admin/login.page.html", &models.TemplateData{Title: "Admin Login"})
}

func (app *Application) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	// --- DEBUG LOGS (Remove these later) ---
	// This will print to your Render console so you can see what is happening
	log.Printf("Login Attempt: User=%s", username)
	// ----------------------------------------

	id, err := app.Users.Authenticate(username, password)
	if err != nil {
		log.Println("Login Failed:", err)
		http.Redirect(w, r, "/admin/login?error=true", http.StatusSeeOther)
		return
	}

	// Login Success
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    fmt.Sprintf("%d", id),
		Path:     "/",
		HttpOnly: true,
		// Secure: true, // Uncomment this line if you are on HTTPS (Render)
	})

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

func (app *Application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "admin_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // Expire immediately
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// requireAdmin checks if the user is logged in
func (app *Application) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for cookie
		_, err := r.Cookie("admin_session")
		if err != nil {
			// No cookie? Kick them to login page
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		// Cookie exists? Let them pass
		next(w, r)
	}
}
func (app *Application) orderConfirmedHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	// Fetch order details to show on the Thank You page
	order, err := app.Orders.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	items, err := app.Orders.GetOrderItems(id)
	if err != nil {
		log.Println("Error fetching order items:", err)
	}

	data := &models.TemplateData{
		Title:      "Order Confirmed",
		Order:      order,
		OrderItems: items,
		WhatsAppURL: whatsAppOrderURL(order, items),
	}

	app.render(w, r, "order_confirmed.page.html", data)
}

// adminWhatsAppNumber returns the bakery's WhatsApp number in wa.me format
// (digits only, with country code). Configure via ADMIN_WHATSAPP.
func adminWhatsAppNumber() string {
	raw := os.Getenv("ADMIN_WHATSAPP")
	if raw == "" {
		raw = "254735792437"
	}
	digits := ""
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			digits += string(c)
		}
	}
	// Local 07... form -> 254...
	if len(digits) == 10 && digits[0] == '0' {
		digits = "254" + digits[1:]
	}
	return digits
}

// whatsAppOrderURL builds a wa.me link with the order summary pre-filled,
// so one tap from the customer delivers the order to the admin's WhatsApp.
func whatsAppOrderURL(order *models.Order, items []repository.OrderDetailItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Hello Crave & Glaze! I just placed an order on the website.\n\n")
	fmt.Fprintf(&b, "Order #%d\n", order.ID)
	fmt.Fprintf(&b, "Name: %s %s\n", order.FirstName, order.LastName)
	fmt.Fprintf(&b, "Phone: %s\n", order.CustomerPhone)
	b.WriteString("Items:\n")
	for _, it := range items {
		fmt.Fprintf(&b, "- %dx %s (%s) - KES %.2f\n", it.Quantity, it.ProductName, it.WeightLabel, it.Price)
		if it.Icing != "" {
			fmt.Fprintf(&b, "  Icing: %s\n", it.Icing)
		}
		if it.Message != "" {
			fmt.Fprintf(&b, "  Message: %s\n", it.Message)
		}
	}
	fmt.Fprintf(&b, "\nTotal: KES %.2f", order.TotalAmount)
	return "https://wa.me/" + adminWhatsAppNumber() + "?text=" + url.QueryEscape(b.String())
}
func (app *Application) searchHandler(w http.ResponseWriter, r *http.Request) {
	// Get the search term from the URL (e.g., /search?q=chocolate)
	query := r.URL.Query().Get("q")

	// Fetch matching products
	products, err := app.Products.Search(query)
	if err != nil {
		log.Println("Search error:", err)
		http.Error(w, "Server Error", 500)
		return
	}

	data := &models.TemplateData{
		Title:    "Search Results: " + query,
		Products: products,
	}

	// We can reuse the category page template since it just lists products
	app.render(w, r, "category.page.html", data)
}
