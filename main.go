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

func handlerAgg(s *state, _ command) error {
	feed, err := rss.FetchFeed("https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	fmt.Println("Title:", feed.Channel.Title)
	fmt.Println("Link :", feed.Channel.Link)
	fmt.Println("Desc :", feed.Channel.Description)
	fmt.Printf("Items: %d\n", len(feed.Channel.Items))
	for i, item := range feed.Channel.Items {
		fmt.Printf("--------Item %d---------\n", i)
		fmt.Println("Title: ", item.Title)
		fmt.Println("Description: ", item.Description)
		fmt.Println("Link: ", item.Link)
		fmt.Println("PubDate: ", item.PubDate)
	}
	return nil
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("%s: usage: addfeed <name> <url>", cmd.name)
	}
	name, url := cmd.args[0], cmd.args[1]

	uid, err := database.GetUserID(s.db, s.cfg.CurrentUser)
	if err != nil {
		return fmt.Errorf("addfeed: %w", err)
	}

	id, err := database.CreateFeed(s.db, name, url, uid)
	if err != nil {
		return err
	}
	fmt.Printf("Feed added : %s → %s\n", name, url)

	ff, err := database.CreateFeedFollow(s.db, uid, id)
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

func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: follow <feed-url>", cmd.name)
	}
	url := cmd.args[0]

	feedID, _, err := database.GetFeedByURL(s.db, url)
	if err != nil {
		return err
	}

	userID, err := database.GetUserID(s.db, s.cfg.CurrentUser)
	if err != nil {
		return err
	}

	ff, err := database.CreateFeedFollow(s.db, userID, feedID)
	if err != nil {
		return err
	}

	fmt.Printf("Followed: %s → %s (by %s)\n",
		ff.FeedName, ff.FeedURL, ff.UserName)
	return nil
}

func handlerFollowing(s *state, _ command) error {
	userID, err := database.GetUserID(s.db, s.cfg.CurrentUser)
	if err != nil {
		return err
	}
	follows, err := database.GetFeedFollowsForUser(s.db, userID)
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

	s := state{cfg: cfg, db: db}
	c := commands{list: make(commandMap)}
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", handlerAddFeed)
	c.register("feeds", handlerFeeds)
	c.register("follow", handlerFollow)
	c.register("following", handlerFollowing)
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
