package database

import (
	"database/sql"
	"fmt"
)

func CreateUserTable(db *sql.DB) error {
	// Create a table
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id   			 INTEGER PRIMARY KEY,
    				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    				name 			 TEXT NOT NULL
        );

    		-- keep updated_at in sync on every UPDATE
  			CREATE TRIGGER IF NOT EXISTS users_updated_at
  			AFTER UPDATE ON users
  			FOR EACH ROW
  			BEGIN
    			UPDATE users
    			SET    updated_at = CURRENT_TIMESTAMP
    			WHERE  id = OLD.id;
  			END;
    `)
	if err != nil {
		return err
	}
	return nil
}

func DropUserTable(db *sql.DB) error {
	// 1) Drop the table & trigger if they exist
	if _, err := db.Exec(`
    DROP TRIGGER IF EXISTS users_updated_at;
    DROP TABLE   IF EXISTS users;
  `); err != nil {
		return fmt.Errorf("reset: failed to drop schema: %w", err)
	}
	return nil
}

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

func ContainsUser(db *sql.DB, username string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE name = ?);`,
		username,
	).Scan(&exists)
	return exists, err
}

func RegisterUser(db *sql.DB, username string) error {
	userFound, err := ContainsUser(db, username)
	if err != nil {
		return err
	}
	if !userFound {
		// Insert the new user
		_, err := db.Exec(
			`INSERT INTO users (name) VALUES (?);`,
			username,
		)
		if err != nil {
			return fmt.Errorf("failed to register %q: %w", username, err)
		}
	} else {
		return fmt.Errorf("failed to register: user \"%s\" already exists!", username)
	}
	return nil
}
