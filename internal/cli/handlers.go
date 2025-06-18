package cli

import (
	"blogo/internal/database"
	"blogo/internal/rss"
	"blogo/internal/utils"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func scrapeFeeds(s *State) {
	feeds, err := database.GetAllFeeds(s.DB)
	if err != nil {
		fmt.Println("scrapeFeeds: could not list feeds:", err)
		return
	}

	for _, ff := range feeds {
		if err := database.MarkFeedFetched(s.DB, ff.ID); err != nil {
			fmt.Println("scrapeFeeds: mark fetched:", err)
			// continue on to the fetch even if marking fails
		}

		feed, err := rss.FetchFeed(ff.URL)
		if err != nil {
			fmt.Printf("failed to fetch %q: %v\n", ff.URL, err)
			continue
		}
		fmt.Printf("=== Feed: %s (%s) ===\n", feed.Channel.Title, ff.URL)
		for _, item := range feed.Channel.Items {
			pub, err := utils.ParsePubDate(item.PubDate)
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
			if err := database.CreatePost(s.DB, post); err != nil {
				fmt.Printf("error saving post %q: %v\n", post.URL, err)
			}
		}
		fmt.Println()
	}
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("%s: usage: login <userName>", cmd.Name)
	}
	userName := cmd.Args[0]
	userFound, err := database.ContainsUser(s.DB, userName)
	if userFound {
		if err = s.Cfg.SetUser(userName); err != nil {
			return err
		}

		fmt.Println("Login Success!")
		return nil
	}
	return fmt.Errorf("%s: User not registered!", cmd.Name)
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("%s: usage: register <userName>", cmd.Name)
	}
	userName := cmd.Args[0]
	if err := database.RegisterUser(s.DB, userName); err != nil {
		return err
	}
	fmt.Println("Registered user:", userName)
	if err := s.Cfg.SetUser(userName); err != nil {
		return err
	}
	return nil
}

func HandlerReset(s *State, _ Command) error {
	if err := database.DropAllTables(s.DB); err != nil {
		return err
	}
	if err := database.CreateUserTable(s.DB); err != nil {
		return err
	}
	if err := database.CreateFeedsTable(s.DB); err != nil {
		return err
	}
	if err := database.CreateFeedFollowsTable(s.DB); err != nil {
		return err
	}
	if err := database.CreatePostsTable(s.DB); err != nil {
		return err
	}

	fmt.Println("Database has been reset to blank State.")
	return nil
}

func HandlerUsers(s *State, _ Command) error {
	users, err := database.GetUsers(s.DB)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user == s.Cfg.CurrentUser {
			fmt.Printf("* %s (current)\n", user)
		} else {
			fmt.Printf("* %s\n", user)
		}
	}
	return nil
}

func HandlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("%s: usage: agg <interval>", cmd.Name)
	}
	// parse “1s”, “1m”, “1h”
	d, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q: %w", cmd.Name, cmd.Args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", d)
	scrapeFeeds(s)

	ticker := time.NewTicker(d)
	for {
		<-ticker.C
		scrapeFeeds(s)
	}
}

func HandlerBrowse(s *State, cmd Command, user database.User) error {
	// default limit = 2
	limit := 2
	if len(cmd.Args) == 1 {
		if l, err := strconv.Atoi(cmd.Args[0]); err == nil && l > 0 {
			limit = l
		} else {
			return fmt.Errorf("%s: invalid limit %q", cmd.Name, cmd.Args[0])
		}
	} else if len(cmd.Args) > 1 {
		return fmt.Errorf("%s: usage: browse [limit]", cmd.Name)
	}

	posts, err := database.GetPostsForUser(s.DB, user.ID, limit)
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
			utils.Truncate(p.Description.String, 100),
		)
	}
	return nil
}

func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("%s: usage: addfeed <Name> <url>", cmd.Name)
	}
	Name, url := cmd.Args[0], cmd.Args[1]

	id, err := database.CreateFeed(s.DB, Name, url, user.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Feed added : %s → %s\n", Name, url)

	ff, err := database.CreateFeedFollow(s.DB, user.ID, id)
	if err != nil {
		return fmt.Errorf("addfeed: failed to auto‑follow: %w", err)
	}
	fmt.Printf("Auto‑followed: %s (id=%d)\n", ff.FeedName, ff.ID)
	return nil
}

func HandlerFeeds(s *State, _ Command) error {
	feeds, err := database.GetFeeds(s.DB)
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

func HandlerFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("%s: usage: follow <feed-url>", cmd.Name)
	}
	url := cmd.Args[0]

	feedID, _, err := database.GetFeedByURL(s.DB, url)
	if err != nil {
		return err
	}

	ff, err := database.CreateFeedFollow(s.DB, user.ID, feedID)
	if err != nil {
		return err
	}

	fmt.Printf("Followed: %s → %s (by %s)\n",
		ff.FeedName, ff.FeedURL, ff.UserName)
	return nil
}

func HandlerFollowing(s *State, _ Command, user database.User) error {
	follows, err := database.GetFeedFollowsForUser(s.DB, user.ID)
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

func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("%s: usage: unfollow <feed-url>", cmd.Name)
	}
	feedURL := cmd.Args[0]

	if err := database.DeleteFeedFollowByUserAndURL(s.DB, user.ID, feedURL); err != nil {
		return fmt.Errorf("%s: %w", cmd.Name, err)
	}

	fmt.Printf("Unfollowed feed %q for user %s\n", feedURL, user.Username)
	return nil
}
