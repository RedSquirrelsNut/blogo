package main

import (
	"blogo/internal/cli"
	"blogo/internal/config"
	"blogo/internal/database"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type App struct {
	Cfg *config.Config
	DB  *sql.DB
}

func Setup() *App {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", "./blogo.db")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

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
	return &App{Cfg: cfg, DB: db}
}

func (app *App) Close() {
	app.DB.Close()
}

func (app *App) Run() {
	s := cli.State{Cfg: app.Cfg, DB: app.DB}
	c := cli.Commands{List: make(cli.CommandMap)}
	cli.RegisterAllCommands(&c)
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Usage: blogo <some-arg>")
		os.Exit(1)
	}

	com := cli.Command{Name: args[0], Args: args[1:]}
	if err := c.Run(&s, com); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func main() {
	app := Setup()
	defer app.Close()
	app.Run()
}
