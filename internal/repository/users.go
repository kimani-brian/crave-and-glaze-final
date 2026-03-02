package repository

import (
	"database/sql"
	"errors"
	"os"
)

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Authenticate(username, password string) (int, error) {
	// 1. Get credentials from Environment Variables (Render/Docker)
	expectedUser := os.Getenv("ADMIN_USERNAME")
	expectedPass := os.Getenv("ADMIN_PASSWORD")

	// 2. Fallback for Localhost (If you forgot to set env vars locally)
	// This ensures you can always login locally with admin/password123
	if expectedUser == "" {
		expectedUser = "admin"
	}
	if expectedPass == "" {
		expectedPass = "password123"
	}

	// 3. Compare Input vs Expected
	if username == expectedUser && password == expectedPass {
		return 1, nil // Return ID 1 for the admin
	}

	return 0, errors.New("invalid credentials")
}
