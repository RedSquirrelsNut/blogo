package database

import (
	"database/sql"
	"fmt"
)

// FeedToFetch represents a minimal feed for fetching operations.
type FeedToFetch struct {
	ID  int64
	URL string
}

// FeedInfo contains feed listing information, including owner username.
type FeedInfo struct {
	Name     string
	URL      string
	Username string
}

// CreateFeed inserts a new feed with the given name, URL, and owner user ID.
// Returns the new feed's ID or an error.
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

// GetFeedByURL returns the feed ID and name for the specified URL.
// Returns an error if the feed is not found.
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

// GetAllFeeds returns all feeds as FeedToFetch, ordered by ID.
// Used for fetch scheduling.
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

// GetFeeds lists all feeds and the user who added each feed.
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

// GetNextFeedToFetch returns the next feed to fetch by fetch time priority.
// Returns nil if no feeds are available.
func GetNextFeedToFetch(db *sql.DB) (*FeedToFetch, error) {
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

// MarkFeedFetched updates the last_fetched_at timestamp for the given feed ID.
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
