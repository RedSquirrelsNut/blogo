package database

import (
	"database/sql"
	"fmt"
)

type FeedInfo struct {
	Name     string
	URL      string
	Username string
}

func GetFeeds(db *sql.DB) ([]FeedInfo, error) {
	rows, err := db.Query(`
        SELECT f.name, f.url, u.name
        FROM feeds AS f
        JOIN users AS u ON f.user_id = u.id
        ORDER BY f.id;
    `)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var out []FeedInfo
	for rows.Next() {
		var fi FeedInfo
		if err := rows.Scan(&fi.Name, &fi.URL, &fi.Username); err != nil {
			return nil, fmt.Errorf("scan feed row: %w", err)
		}
		out = append(out, fi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}
	return out, nil
}

func CreateFeedsTable(db *sql.DB) error {
	if _, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS feeds (
  				id          INTEGER PRIMARY KEY,
  				created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  				updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  				name        TEXT    NOT NULL,
  				url         TEXT    NOT NULL UNIQUE,
  				user_id     INTEGER NOT NULL,
  				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				);

				CREATE TRIGGER IF NOT EXISTS feeds_updated_at
				AFTER UPDATE ON feeds
				FOR EACH ROW
				BEGIN
  				UPDATE feeds
    				SET updated_at = CURRENT_TIMESTAMP
    				WHERE id = OLD.id;
				END;
    `); err != nil {
		return err
	}
	return nil
}

func CreateFeed(db *sql.DB, name, url string, userID int64) error {
	_, err := db.Exec(
		`INSERT INTO feeds (name, url, user_id) VALUES (?, ?, ?);`,
		name, url, userID,
	)
	if err != nil {
		return fmt.Errorf("create feed %q: %w", name, err)
	}
	return nil
}

func CreateUserTable(db *sql.DB) error {
	if _, err := db.Exec(`
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
    `); err != nil {
		return err
	}
	return nil
}

func DropFeedsTable(db *sql.DB) error {
	if _, err := db.Exec(`
		DROP TRIGGER IF EXISTS feeds_updated_at;
    DROP TABLE   IF EXISTS feeds;
  `); err != nil {
		return fmt.Errorf("reset: failed to drop schema: %w", err)
	}
	return nil
}

func DropUserTable(db *sql.DB) error {
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
		if _, err := db.Exec(
			`INSERT INTO users (name) VALUES (?);`,
			username,
		); err != nil {
			return fmt.Errorf("failed to register %q: %w", username, err)
		}
	} else {
		return fmt.Errorf("failed to register: user \"%s\" already exists!", username)
	}
	return nil
}
