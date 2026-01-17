package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	// 2. Get DB variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		user, password, dbname, host, port)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 3. Get Admin Credentials from ENV (The Secure Way)
	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	// Safety check: Don't run if variables are missing
	if adminUser == "" || adminPass == "" {
		log.Fatal("Error: ADMIN_USERNAME or ADMIN_PASSWORD not set in .env file")
	}

	// 4. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPass), 12)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Insert
	stmt := `
    INSERT INTO users (username, password_hash) 
    VALUES ($1, $2) 
    ON CONFLICT (username) 
    DO UPDATE SET password_hash = EXCLUDED.password_hash;
	`
	_, err = db.Exec(stmt, adminUser, string(hashedPassword))
	if err != nil {
		log.Fatal("Error seeding admin:", err)
	}

	fmt.Println("--------------------------------------")
	fmt.Println("Admin user seeded successfully!")
	fmt.Printf("Username: %s\n", adminUser)
	fmt.Println("Password: [HIDDEN] (Read from .env)")
	fmt.Println("--------------------------------------")
}
