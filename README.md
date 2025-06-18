# Blogo

A simple CLI-based RSS aggregator for multiple users, written in Go.  
Supports user registration, following feeds, fetching posts, and browsing.  
Made to learn Go/Sqlite.

## Features

- Register/login users
- Add RSS feeds per user
- Follow/unfollow feeds
- Periodic feed scraping
- Browse user-specific posts

## Project Structure

- `main.go` - Entry point
- `internal/cli/` - CLI command handling/setup
- `internal/config/` - Config reading/writing
- `internal/rss/` - RSS feed fetching/parsing
- `internal/database/` - Schema/Database logic, split by concern: users, feeds, posts, feed follows
- `internal/utils/` - Utility functions (e.g. date parsing, string truncation, RSS-specific helpers, HTML cleanup)

## Testing (WIP)

All core logic is tested using in-memory SQLite. Run all tests:

```sh
go test ./internal/...
```

## Usage

```sh
go build -o blogo
# examples of how to create user, login to a user, and add a feed.
./blogo register alice
./blogo login alice
./blogo addfeed "My Blog" https://myblog.com/rss
# run in the background
./blogo agg 5m
# display 5 most recent posts
./blogo browse 5
```
### Available Commands 
- `register *username*` - Create a user
- `login *username*` - Login as user
- `agg *interval*` - Runs aggregator, fetching every interval
- `users` - List all users
- `feeds` - List all feeds
#### Login Required
- `addfeed *name* *url*` - Add feed, auto follow
- `follow *url*` - Follows a feed
- `unfollow *url*` - Unfollows a feed
- `following` - Lists all feeds followed by current user
- `browse *?num*` - Displays most recent posts (last 2 with no arg)


## Documentation (WIP)

All exported functions and types are documented with GoDoc comments.  
See code for further inline explanations.
