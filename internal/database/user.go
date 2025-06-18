package database

import (
	"database/sql"
	"fmt"
)

type User struct {
	ID       int64
	Username string
}

// RegisterUser adds a user to the database if they do not already exist.
//
// Returns an error if the user already exists or if the insert fails.
func RegisterUser(db *sql.DB, username string) error {
	exists, err := ContainsUser(db, username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("failed to register: user %q already exists", username)
	}
	_, err = db.Exec(`INSERT INTO users (name) VALUES (?);`, username)
	if err != nil {
		return fmt.Errorf("failed to register %q: %w", username, err)
	}
	return nil
}

// GetUsers returns all usernames in the database.
//
// Returns a slice of usernames, or an error if the query fails.
func GetUsers(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		users = append(users, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// ContainsUser checks if a user with the given username exists.
//
// Returns true if the user exists, false otherwise. Returns an error on query failure.
func ContainsUser(db *sql.DB, username string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE name = ?);`,
		username,
	).Scan(&exists)
	return exists, err
}

// GetUserID fetches the user ID for the given username.
//
// Returns the user ID, or an error if the user does not exist or query fails.
func GetUserID(db *sql.DB, username string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM users WHERE name = ?;`,
		username,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("user %q not found", username)
	}
	return id, err
}
