package database

import (
	"blogo/internal/database/schema"
	"database/sql"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int64
	Username string
}

type Post struct {
	ID          int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Title       string
	URL         string
	Description sql.NullString
	PublishedAt sql.NullTime
	FeedID      int64
}

// CreatePost inserts a new post record.
// Returns ErrConstraint if the URL already exists.
func CreatePost(db *sql.DB, p *Post) error {
	const q = `
      INSERT INTO posts (title, url, description, published_at, feed_id)
      VALUES (?, ?, ?, ?, ?);
    `
	if _, err := db.Exec(q,
		p.Title,
		p.URL,
		p.Description,
		p.PublishedAt,
		p.FeedID,
	); err != nil {
		// if it's a sqlite3 unique‑constraint on posts.url, ignore it:
		if sqlErr, ok := err.(sqlite3.Error); ok {
			if sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil
			}
		}
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

// GetPostsForUser returns the most-recent N posts from all feeds the user follows.
func GetPostsForUser(db *sql.DB, userID int64, limit int) ([]Post, error) {
	const q = `
      SELECT p.id, p.created_at, p.updated_at,
             p.title, p.url, p.description, p.published_at, p.feed_id
      FROM posts AS p
      JOIN feed_follows AS ff ON ff.feed_id = p.feed_id
      WHERE ff.user_id = ?
      ORDER BY p.published_at DESC
      LIMIT ?;
    `
	rows, err := db.Query(q, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(
			&p.ID, &p.CreatedAt, &p.UpdatedAt,
			&p.Title, &p.URL, &p.Description, &p.PublishedAt, &p.FeedID,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func CreatePostsTable(db *sql.DB) error {
	schema, err := schema.LoadSchema("posts")
	if err != nil {
		return err
	}

	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return nil
}

type FeedToFetch struct {
	ID  int64
	URL string
}

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

func GetAllFeeds(db *sql.DB) ([]FeedToFetch, error) {
	const q = `SELECT id, url FROM feeds ORDER BY id;`
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("get all feeds: %w", err)
	}
	defer rows.Close()

	var out []FeedToFetch
	for rows.Next() {
		var f FeedToFetch
		if err := rows.Scan(&f.ID, &f.URL); err != nil {
			return nil, fmt.Errorf("scan feed row: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}
	return out, nil
}

func GetNextFeedToFetch(db *sql.DB) (*FeedToFetch, error) {
	// NULLS FIRST means feeds never fetched sort before those with timestamps
	const q = `
      SELECT id, url
      FROM feeds
      ORDER BY last_fetched_at IS NOT NULL, last_fetched_at ASC
      LIMIT 1;
    `
	row := db.QueryRow(q)
	var f FeedToFetch
	if err := row.Scan(&f.ID, &f.URL); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no feeds to fetch")
		}
		return nil, err
	}
	return &f, nil
}

func MarkFeedFetched(db *sql.DB, feedID int64) error {
	const q = `
      UPDATE feeds
      SET last_fetched_at = CURRENT_TIMESTAMP,
          updated_at      = CURRENT_TIMESTAMP
      WHERE id = ?;
    `
	_, err := db.Exec(q, feedID)
	return err
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

func DeleteFeedFollowByUserAndURL(db *sql.DB, userID int64, feedURL string) error {
	var feedID int64
	err := db.QueryRow(
		`SELECT id FROM feeds WHERE url = ?;`,
		feedURL,
	).Scan(&feedID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("feed %q not found", feedURL)
	}
	if err != nil {
		return fmt.Errorf("lookup feed ID: %w", err)
	}

	res, err := db.Exec(
		`DELETE FROM feed_follows WHERE user_id = ? AND feed_id = ?;`,
		userID, feedID,
	)
	if err != nil {
		return fmt.Errorf("delete follow: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete count: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("no follow found for user %d on feed %q", userID, feedURL)
	}
	return nil
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

func DropAllTables(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF;`); err != nil {
		return err
	}

	rows, err := db.Query(`
        SELECT name
        FROM sqlite_master
        WHERE type='table'
          AND name NOT LIKE 'sqlite_%';
    `)
	if err != nil {
		return err
	}

	var tables []string
	for rows.Next() {
		var tbl string
		if err := rows.Scan(&tbl); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, tbl)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, tbl := range tables {
		if _, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, tbl)); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return err
	}

	return nil
}
