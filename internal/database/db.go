package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error
	var connStr string

	// ==========================================
	// 1. DETECT ENVIRONMENT (Render or Local)
	// ==========================================

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL != "" {
		// Running on Render (Production)
		connStr = dbURL
		fmt.Println("Using Render DATABASE_URL")
	} else {
		// Running Locally / Docker
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}

		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}

		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "admin"
		}

		dbPass := os.Getenv("DB_PASSWORD")
		if dbPass == "" {
			dbPass = "password123"
		}

		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "crave_glaze"
		}

		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPass, dbName,
		)

		fmt.Println("Using Local Database Config")
	}

	// ==========================================
	// 2. CONNECT TO DATABASE
	// ==========================================

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging database:", err)
	}

	fmt.Println("Successfully connected to Database!")

	// ==========================================
	// 3. RUN SCHEMA (Create Tables If Missing)
	// ==========================================

	fmt.Println("Running schema.sql...")

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Printf("Warning: Could not read schema.sql: %v\n", err)
	} else {
		_, err = DB.Exec(string(schema))
		if err != nil {
			log.Printf("Schema execution warning: %v\n", err)
		} else {
			fmt.Println("Schema executed successfully.")
		}
	}

	// ==========================================
	// 4. RUN PRODUCTION MIGRATIONS (IMPORTANT)
	// ==========================================

	log.Println("Checking for missing columns...")

	migrations := []string{
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS first_name VARCHAR(100);",
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS last_name VARCHAR(100);",
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS email VARCHAR(150);",
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS whatsapp_number VARCHAR(50);",
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS mpesa_receipt VARCHAR(50);",
		"ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(20);",
	}

	for _, query := range migrations {
		_, err := DB.Exec(query)
		if err != nil {
			// Ignore harmless "already exists" type errors
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("Migration Warning: %v\n", err)
			}
		}
	}

	log.Println("Database Migrations Complete.")
}
