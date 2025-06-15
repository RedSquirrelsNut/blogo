package database

import (
	"blogo/internal/database/schema"
	"database/sql"
	"fmt"
	"time"
)

type FeedFollowInfo struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64
	FeedID    int64
	UserName  string
	FeedName  string
	FeedURL   string
}

const createFeedFollowSQL = `
WITH ins AS (
  INSERT INTO feed_follows (user_id, feed_id)
  VALUES (?, ?)
  RETURNING id, created_at, updated_at, user_id, feed_id
)
SELECT
  ins.id,
  ins.created_at,
  ins.updated_at,
  ins.user_id,
  ins.feed_id,
  u.name   AS user_name,
  f.name   AS feed_name,
  f.url    AS feed_url
FROM ins
JOIN users AS u ON u.id = ins.user_id
JOIN feeds AS f ON f.id = ins.feed_id;
`

// CreateFeedFollow inserts a new follow and returns the full record.
func CreateFeedFollow(db *sql.DB, userID, feedID int64) (*FeedFollowInfo, error) {
	row := db.QueryRow(createFeedFollowSQL, userID, feedID)

	var ff FeedFollowInfo
	if err := row.Scan(
		&ff.ID,
		&ff.CreatedAt,
		&ff.UpdatedAt,
		&ff.UserID,
		&ff.FeedID,
		&ff.UserName,
		&ff.FeedName,
		&ff.FeedURL,
	); err != nil {
		return nil, fmt.Errorf("create feed_follow: %w", err)
	}
	return &ff, nil
}

func CreateFeedFollowsTable(db *sql.DB) error {
	schema, err := schema.LoadSchema("feed_follows")
	if err != nil {
		return err
	}

	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return nil
}

func DropFeedFollows(db *sql.DB) error {
	if _, err := db.Exec(`
  	DROP TRIGGER IF EXISTS feed_follows_updated_at;
		DROP TABLE   IF EXISTS feed_follows;
  `); err != nil {
		return fmt.Errorf("reset: failed to drop schema: %w", err)
	}
	return nil
}

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
	schema, err := schema.LoadSchema("feeds")
	if err != nil {
		return err
	}

	if _, err := db.Exec(schema); err != nil {
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
	schema, err := schema.LoadSchema("users")
	if err != nil {
		return err
	}

	if _, err := db.Exec(schema); err != nil {
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
