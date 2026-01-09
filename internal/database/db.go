package database

import (
	"database/sql"
	"fmt"
	"log"
	"os" // <--- Add this

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error

	// Read from Environment Variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		user, password, dbname, host, port)

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	fmt.Println("Successfully connected to Database!")
}
