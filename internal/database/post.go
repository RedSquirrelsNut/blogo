package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Post represents a single post in the database.
type Post struct {
	ID          int64          // Post ID
	CreatedAt   time.Time      // Time of creation
	UpdatedAt   time.Time      // Last update time
	Title       string         // Post title
	URL         string         // Post URL
	Description sql.NullString // Post description (nullable)
	PublishedAt sql.NullTime   // Time published (nullable)
	FeedID      int64          // Associated feed ID
}

// CreatePost inserts a new post into the posts table.
//
// Returns an error if the insert fails.
func CreatePost(db *sql.DB, p *Post) error {
	_, err := db.Exec(
		`INSERT INTO posts (title, url, description, published_at, feed_id) VALUES (?, ?, ?, ?, ?) ON CONFLICT(url) DO NOTHING;`,
		p.Title, p.URL, p.Description, p.PublishedAt, p.FeedID,
	)
	if err != nil {
		return fmt.Errorf("create post %q: %w", p.Title, err)
	}
	return nil
}

// GetPostsForUser returns the latest posts for all feeds followed by a user.
//
// The limit parameter restricts the number of posts returned.
// Returns a slice of posts, or an error if the query fails.
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
		return nil, fmt.Errorf("get posts for user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(
			&p.ID, &p.CreatedAt, &p.UpdatedAt,
			&p.Title, &p.URL, &p.Description, &p.PublishedAt, &p.FeedID,
		); err != nil {
			return nil, fmt.Errorf("scan post row: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return out, nil
}
