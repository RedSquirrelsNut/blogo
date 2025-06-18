package database

import (
	"database/sql"
	"fmt"
	"time"
)

// FeedFollowInfo represents a feed follow relationship, including user and feed details.
type FeedFollowInfo struct {
	ID        int64     // Follow record ID
	CreatedAt time.Time // Time of creation
	UpdatedAt time.Time // Last update time
	UserID    int64     // User ID
	FeedID    int64     // Feed ID
	UserName  string    // User's name
	FeedName  string    // Feed's name
	FeedURL   string    // Feed's URL
}

// CreateFeedFollow creates a feed follow relationship for a user and feed.
//
// Returns the full FeedFollowInfo for the new follow, or an error.
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

// DeleteFeedFollowByUserAndURL deletes a feed follow entry for a user and a feed URL.
//
// Returns an error if the feed or follow is not found, or on database error.
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

// GetFeedFollowsForUser returns all feed follows for a given user, with joined feed/user details.
//
// Returns a slice of FeedFollowInfo, or an error if the query fails.
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
