package repository

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Authenticate(username, password string) (int, error) {
	var id int
	var hashedPassword string

	stmt := `SELECT id, password_hash FROM users WHERE username = $1`
	row := m.DB.QueryRow(stmt, username)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("invalid credentials")
		}
		return 0, err
	}

	// Compare the stored hash with the password provided
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return 0, errors.New("invalid credentials")
	}

	return id, nil
}
