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

func CreateFeedFollow(db *sql.DB, userID, feedID int64) (*FeedFollowInfo, error) {
	res, err := db.Exec(
		`INSERT INTO feed_follows (user_id, feed_id) VALUES (?, ?);`,
		userID, feedID,
	)
	if err != nil {
		return nil, fmt.Errorf("create feed_follow: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create feed_follow: unable to fetch new ID: %w", err)
	}

	row := db.QueryRow(`
        SELECT ff.id, ff.created_at, ff.updated_at,
               ff.user_id, ff.feed_id,
               u.name AS user_name,
               f.name AS feed_name, f.url AS feed_url
        FROM feed_follows AS ff
        JOIN users AS u ON u.id = ff.user_id
        JOIN feeds AS f ON f.id = ff.feed_id
        WHERE ff.id = ?;
    `, id)

	var ff FeedFollowInfo
	if err := row.Scan(
		&ff.ID, &ff.CreatedAt, &ff.UpdatedAt,
		&ff.UserID, &ff.FeedID,
		&ff.UserName, &ff.FeedName, &ff.FeedURL,
	); err != nil {
		return nil, fmt.Errorf("fetch created feed_follow: %w", err)
	}
	return &ff, nil
}

func GetFeedByURL(db *sql.DB, url string) (id int64, name string, err error) {
	err = db.QueryRow(
		`SELECT id, name FROM feeds WHERE url = ?;`,
		url,
	).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("feed %q not found", url)
	}
	return id, name, err
}

func GetFeedFollowsForUser(db *sql.DB, userID int64) ([]FeedFollowInfo, error) {
	rows, err := db.Query(`
        SELECT ff.id, ff.created_at, ff.updated_at,
               ff.user_id, ff.feed_id,
               u.name AS user_name,
               f.name AS feed_name, f.url AS feed_url
        FROM feed_follows AS ff
        JOIN users AS u ON u.id = ff.user_id
        JOIN feeds AS f ON f.id = ff.feed_id
        WHERE ff.user_id = ?
        ORDER BY ff.id;
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("query feed_follows: %w", err)
	}
	defer rows.Close()

	var out []FeedFollowInfo
	for rows.Next() {
		var ff FeedFollowInfo
		if err := rows.Scan(
			&ff.ID, &ff.CreatedAt, &ff.UpdatedAt,
			&ff.UserID, &ff.FeedID,
			&ff.UserName, &ff.FeedName, &ff.FeedURL,
		); err != nil {
			return nil, fmt.Errorf("scan feed_follow: %w", err)
		}
		out = append(out, ff)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed_follows: %w", err)
	}
	return out, nil
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

func CreateFeed(db *sql.DB, name, url string, userID int64) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO feeds (name, url, user_id) VALUES (?, ?, ?);`,
		name, url, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("create feed %q: %w", url, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("retrieve new feed ID: %w", err)
	}
	return id, nil
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
