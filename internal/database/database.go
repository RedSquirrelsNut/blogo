package database

import (
	"database/sql"
	"embed"
	"fmt"
)

//go:embed schema/users.sql
//go:embed schema/feeds.sql
//go:embed schema/feed_follows.sql
//go:embed schema/posts.sql
var ddlFiles embed.FS

func LoadSQL(name string) (string, error) {
	b, err := ddlFiles.ReadFile("schema/" + name + ".sql")
	if err != nil {
		return "", fmt.Errorf("load Schema: %w", err)
	}
	return string(b), nil
}

// CreateTable creates a table in the database using the given schema name.
//
// It loads the schema SQL using schema.LoadSQL and executes it.
// Returns an error if loading or executing the schema fails.
func CreateTable(db *sql.DB, tableName string) error {
	s, err := LoadSQL(tableName)
	if err != nil {
		return fmt.Errorf("load schema for table %s: %w", tableName, err)
	}
	if _, err := db.Exec(s); err != nil {
		return fmt.Errorf("create table %s: %w", tableName, err)
	}
	return nil
}

// DropTable drops the table and its trigger (if provided) from the database.
//
// If triggerName is not empty, the corresponding trigger is dropped as well.
// Returns an error if dropping the trigger or table fails.
func DropTable(db *sql.DB, tableName, triggerName string) error {
	if triggerName != "" {
		if _, err := db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s;`, triggerName)); err != nil {
			return fmt.Errorf("drop trigger %s: %w", triggerName, err)
		}
	}
	if _, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, tableName)); err != nil {
		return fmt.Errorf("drop table %s: %w", tableName, err)
	}
	return nil
}

// DropAllTables drops all user-defined tables in the database, excluding SQLite system tables.
//
// This disables foreign keys, drops all tables, then re-enables foreign keys.
// Returns an error if any operation fails.
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
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var tbl string
		if err := rows.Scan(&tbl); err != nil {
			return err
		}
		tables = append(tables, tbl)
	}
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

// CreateUserTable creates the user table using its schema.
func CreateUserTable(db *sql.DB) error { return CreateTable(db, "users") }

// CreateFeedsTable creates the feeds table using its schema.
func CreateFeedsTable(db *sql.DB) error { return CreateTable(db, "feeds") }

// CreateFeedFollowsTable creates the feed_follows table using its schema.
func CreateFeedFollowsTable(db *sql.DB) error { return CreateTable(db, "feed_follows") }

// CreatePostsTable creates the posts table using its schema.
func CreatePostsTable(db *sql.DB) error { return CreateTable(db, "posts") }

// DropUserTable drops the users table and its trigger.
func DropUserTable(db *sql.DB) error { return DropTable(db, "users", "users_updated_at") }

// DropFeedsTable drops the feeds table and its trigger.
func DropFeedsTable(db *sql.DB) error { return DropTable(db, "feeds", "feeds_updated_at") }

// DropFeedFollowsTable drops the feed_follows table and its trigger.
func DropFeedFollowsTable(db *sql.DB) error {
	return DropTable(db, "feed_follows", "feed_follows_updated_at")
}

// DropPostsTable drops the posts table and its trigger.
func DropPostsTable(db *sql.DB) error { return DropTable(db, "posts", "posts_updated_at") }
