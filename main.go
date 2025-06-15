package main

import (
	"blogo/internal/config"
	"blogo/internal/database"
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
	if f, ok := c.list[cmd.name]; !ok {
		return errors.New("Command " + cmd.name + " not found!")
	} else {
		return f(s, cmd)
	}
}

func (c *commands) register(name string, f commandHandler) {
	c.list[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("%s: usage: login <username>", cmd.name)
	}
	username := cmd.args[0]
	if userFound, err := database.ContainsUser(s.db, username); err != nil {
		return err
	} else if userFound {
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
	if err := database.DropUserTable(s.db); err != nil {
		return err
	}
	if err := database.CreateUserTable(s.db); err != nil {
		return err
	}
	fmt.Println("Database has been reset to blank state.")
	return nil
}

func handlerUsers(s *state, _ command) error {
	if users, err := database.GetUsers(s.db); err != nil {
		return err
	} else {
		for _, user := range users {
			if user == s.cfg.CurrentUser {
				fmt.Printf("* %s (current)\n", user)
			} else {
				fmt.Printf("* %s\n", user)
			}
		}
	}
	return nil
}

func main() {
	if cfg, err := config.Read(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	} else {
		if db, err := sql.Open("sqlite3", "./users.db"); err != nil {
			log.Fatal(err)
			os.Exit(1)
		} else {
			defer db.Close()
			database.CreateUserTable(db)
			s := state{cfg: cfg, db: db}
			c := commands{list: make(commandMap)}
			c.register("login", handlerLogin)
			c.register("register", handlerRegister)
			c.register("reset", handlerReset)
			c.register("users", handlerUsers)
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
	}
}
