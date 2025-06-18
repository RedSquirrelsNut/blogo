# Blogo

A simple CLI-based RSS aggregator for multiple users, written in Go.  
Supports user registration, following feeds, fetching posts, and browsing.

## Features

- Register/login users
- Add RSS feeds per user
- Follow/unfollow feeds
- Periodic feed scraping
- Browse user-specific posts

## Project Structure

- `main.go` - Entry point, CLI command handling, setup
- `internal/database/` - Database logic, split by concern: users, feeds, posts, feed follows
- `internal/utils/` - Utility functions (e.g. date parsing, string truncation)
- `internal/rss/` - RSS-specific helpers, HTML cleanup

## Testing (WIP)

All core logic is tested using in-memory SQLite. Run all tests:

```sh
go test ./internal/...
```

## Usage

```sh
go build -o blogo
./blogo register alice
./blogo login alice
./blogo addfeed "My Blog" https://myblog.com/rss
./blogo agg 5m
./blogo browse
```

## Documentation (WIP)

All exported functions and types are documented with GoDoc comments.  
See code for further inline explanations.
