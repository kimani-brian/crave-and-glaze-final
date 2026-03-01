package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error
	var connStr string

	// 1. Check if we are on Render (They provide DATABASE_URL)
	dbURL := os.Getenv("DATABASE_URL")

	if dbURL != "" {
		// We are on Render! Use the provided URL.
		connStr = dbURL
	} else {
		// 2. We are on Localhost / Docker Compose
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

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPass, dbName)
	}

	// Connect
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	fmt.Println("Successfully connected to Database!")
}
