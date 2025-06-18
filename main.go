package main

import (
	"blogo/internal/config"
	"blogo/internal/database"
	"blogo/internal/rss"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type state struct {
	cfg *config.Config
	db  *sql.DB
}

type command struct {
	name string
	args []string
}

type commandHandler = func(*state, command) error
type commandMap = map[string]commandHandler

type commands struct {
	list commandMap
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.list[cmd.name]
	if !ok {
		return errors.New("Command " + cmd.name + " not found!")
	}
	return f(s, cmd)
}

func (c *commands) register(name string, f commandHandler) {
	c.list[name] = f
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		username := s.cfg.CurrentUser
		if username == "" {
			fmt.Printf("%s: you must login first (try `login <username>`)\n", cmd.name)
			return nil
		}

		uid, err := database.GetUserID(s.db, username)
		if err != nil {
			return fmt.Errorf("%s: could not fetch user record: %w", cmd.name, err)
		}
		user := database.User{ID: uid, Username: username}
		return handler(s, cmd, user)
	}
}
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

var rssDateLayouts = []string{
	time.RFC1123Z,               // “Mon, 02 Jan 2006 15:04:05 -0700”
	time.RFC1123,                // “Mon, 02 Jan 2006 15:04:05 MST”
	time.RFC822Z,                // “02 Jan 06 15:04 -0700”
	time.RFC822,                 // “02 Jan 06 15:04 MST”
	time.RFC850,                 // “Monday, 02-Jan-06 15:04:05 MST”
	time.RFC3339,                // “2006-01-02T15:04:05Z07:00”
	"2006-01-02 15:04:05 -0700", // some feeds use a space‑separated variant
	time.ANSIC,                  // “Mon Jan _2 15:04:05 2006”
}

// parsePubDate attempts each known layout until one succeeds.
func parsePubDate(raw string) (time.Time, error) {
	var lastErr error
	for _, layout := range rssDateLayouts {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("unrecognized date format %q: %v", raw, lastErr)
}

func scrapeFeeds(s *state) {
	// fetch the full list of feeds up front
	feeds, err := database.GetAllFeeds(s.db) // you’ll write this in a moment
	if err != nil {
		fmt.Println("scrapeFeeds: could not list feeds:", err)
		return
	}

	for _, ff := range feeds {
		// mark it so we won’t refetch too soon
		if err := database.MarkFeedFetched(s.db, ff.ID); err != nil {
			fmt.Println("scrapeFeeds: mark fetched:", err)
			// continue on to the fetch even if marking fails
		}

		// fetch & print
		feed, err := rss.FetchFeed(ff.URL)
		if err != nil {
			fmt.Printf("failed to fetch %q: %v\n", ff.URL, err)
			continue
		}
		fmt.Printf("=== Feed: %s (%s) ===\n", feed.Channel.Title, ff.URL)
		for _, item := range feed.Channel.Items {
			pub, err := parsePubDate(item.PubDate)
			if err != nil {
				fmt.Printf("warning: could not parse date %q: %v\n", item.PubDate, err)
			}
			post := &database.Post{
				Title:       item.Title,
				URL:         item.Link,
				Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
				PublishedAt: sql.NullTime{Time: pub, Valid: !pub.IsZero()},
				FeedID:      ff.ID,
			}
			if err := database.CreatePost(s.db, post); err != nil {
				fmt.Printf("error saving post %q: %v\n", post.URL, err)
			}
		}
		fmt.Println()
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: login <username>", cmd.name)
	}
	username := cmd.args[0]
	userFound, err := database.ContainsUser(s.db, username)
	if userFound {
		if err = s.cfg.SetUser(username); err != nil {
			return err
		}

		fmt.Println("Login Success!")
		return nil
	}
	return fmt.Errorf("%s: User not registered!", cmd.name)
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: register <username>", cmd.name)
	}
	username := cmd.args[0]
	if err := database.RegisterUser(s.db, username); err != nil {
		return err
	}
	fmt.Println("Registered user:", username)
	if err := s.cfg.SetUser(username); err != nil {
		return err
	}
	return nil
}

func handlerReset(s *state, _ command) error {
	if err := database.DropAllTables(s.db); err != nil {
		return err
	}
	if err := database.CreateUserTable(s.db); err != nil {
		return err
	}
	if err := database.CreateFeedsTable(s.db); err != nil {
		return err
	}
	if err := database.CreateFeedFollowsTable(s.db); err != nil {
		return err
	}
	if err := database.CreatePostsTable(s.db); err != nil {
		return err
	}

	fmt.Println("Database has been reset to blank state.")
	return nil
}

func handlerUsers(s *state, _ command) error {
	users, err := database.GetUsers(s.db)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user == s.cfg.CurrentUser {
			fmt.Printf("* %s (current)\n", user)
		} else {
			fmt.Printf("* %s\n", user)
		}
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: agg <interval>", cmd.name)
	}
	// parse “1s”, “1m”, “1h”
	d, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q: %w", cmd.name, cmd.args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", d)
	scrapeFeeds(s)

	ticker := time.NewTicker(d)
	for {
		<-ticker.C
		scrapeFeeds(s)
	}
	// unreachable, signature demands an error return
	// return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	// default limit = 2
	limit := 2
	if len(cmd.args) == 1 {
		if l, err := strconv.Atoi(cmd.args[0]); err == nil && l > 0 {
			limit = l
		} else {
			return fmt.Errorf("%s: invalid limit %q", cmd.name, cmd.args[0])
		}
	} else if len(cmd.args) > 1 {
		return fmt.Errorf("%s: usage: browse [limit]", cmd.name)
	}

	posts, err := database.GetPostsForUser(s.db, user.ID, limit)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		fmt.Println("No posts available.")
		return nil
	}
	for i, p := range posts {
		fmt.Printf("===========================Post %d============================\n", i+1)
		fmt.Printf("• %s (%s)\n  published: %s\n  %s\n\n",
			p.Title, p.URL,
			p.PublishedAt.Time.Format(time.RFC1123),
			truncate(p.Description.String, 100),
		)
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("%s: usage: addfeed <name> <url>", cmd.name)
	}
	name, url := cmd.args[0], cmd.args[1]

	id, err := database.CreateFeed(s.db, name, url, user.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Feed added : %s → %s\n", name, url)

	ff, err := database.CreateFeedFollow(s.db, user.ID, id)
	if err != nil {
		return fmt.Errorf("addfeed: failed to auto‑follow: %w", err)
	}
	fmt.Printf("Auto‑followed: %s (id=%d)\n", ff.FeedName, ff.ID)
	return nil
}

func handlerFeeds(s *state, _ command) error {
	feeds, err := database.GetFeeds(s.db)
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, f := range feeds {
		fmt.Printf("%s → %s (added by %s)\n", f.Name, f.URL, f.Username)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: follow <feed-url>", cmd.name)
	}
	url := cmd.args[0]

	feedID, _, err := database.GetFeedByURL(s.db, url)
	if err != nil {
		return err
	}

	ff, err := database.CreateFeedFollow(s.db, user.ID, feedID)
	if err != nil {
		return err
	}

	fmt.Printf("Followed: %s → %s (by %s)\n",
		ff.FeedName, ff.FeedURL, ff.UserName)
	return nil
}

func handlerFollowing(s *state, _ command, user database.User) error {
	follows, err := database.GetFeedFollowsForUser(s.db, user.ID)
	if err != nil {
		return err
	}
	if len(follows) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}
	for _, ff := range follows {
		fmt.Printf("- %s (%s)\n", ff.FeedName, ff.FeedURL)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: unfollow <feed-url>", cmd.name)
	}
	feedURL := cmd.args[0]

	if err := database.DeleteFeedFollowByUserAndURL(s.db, user.ID, feedURL); err != nil {
		return fmt.Errorf("%s: %w", cmd.name, err)
	}

	fmt.Printf("Unfollowed feed %q for user %s\n", feedURL, user.Username)
	return nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", "./users.db")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.CreateUserTable(db); err != nil {
		log.Fatal(err)
	}
	if err := database.CreateFeedsTable(db); err != nil {
		log.Fatal(err)
	}
	if err := database.CreateFeedFollowsTable(db); err != nil {
		log.Fatal(err)
	}
	if err := database.CreatePostsTable(db); err != nil {
		log.Fatal(err)
	}

	s := state{cfg: cfg, db: db}
	c := commands{list: make(commandMap)}
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("browse", middlewareLoggedIn(handlerBrowse))
	c.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	c.register("feeds", handlerFeeds)
	c.register("follow", middlewareLoggedIn(handlerFollow))
	c.register("following", middlewareLoggedIn(handlerFollowing))
	c.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Usage: blogo <some-arg>")
		os.Exit(1)
	}

	com := command{name: args[0], args: args[1:]}
	if err := c.run(&s, com); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
