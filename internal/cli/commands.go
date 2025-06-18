package cli

import (
	"blogo/internal/config"
	"blogo/internal/database"
	"database/sql"
	"errors"
	"fmt"
)

type State struct {
	Cfg *config.Config
	DB  *sql.DB
}

type Command struct {
	Name string
	Args []string
}

type CommandHandler = func(*State, Command) error
type CommandMap map[string]CommandHandler

type Commands struct {
	List CommandMap
}

func (c *Commands) Run(s *State, cmd Command) error {
	f, ok := c.List[cmd.Name]
	if !ok {
		return errors.New("Command " + cmd.Name + " not found!")
	}
	return f(s, cmd)
}

func (c *Commands) Register(name string, f CommandHandler) {
	c.List[name] = f
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		username := s.Cfg.CurrentUser
		if username == "" {
			fmt.Printf("%s: you must login first (try `login <username>`)\n", cmd.Name)
			return nil
		}

		uid, err := database.GetUserID(s.DB, username)
		if err != nil {
			return fmt.Errorf("%s: could not fetch user record: %w", cmd.Name, err)
		}
		user := database.User{ID: uid, Username: username}
		return handler(s, cmd, user)
	}
}
