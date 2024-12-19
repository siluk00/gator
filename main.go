package main

import (
	"database/sql"
	"gator/internal/config"
	"gator/internal/database"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	var s state
	var err error
	s.cfg, err = config.Read()

	if err != nil {
		log.Fatalf("Error reading config file: %s\n", err)
	}

	dburl := s.cfg.DbURL
	db, err := sql.Open("postgres", dburl)

	if err != nil {
		log.Fatalf("Error opening database connection: %s\n", err)
	}

	dbQueries := database.New(db)
	s.db = dbQueries

	var cmds commands
	cmds.mapOfCmds = make(map[string]func(*state, command) error)
	cmdsPtr := &cmds
	cmdsPtr.register("login", handlerLogin)
	cmdsPtr.register("register", handlerRegister)
	cmdsPtr.register("reset", handlerReset)
	cmdsPtr.register("users", handleUsers)
	cmdsPtr.register("agg", handleAgg)
	cmdsPtr.register("addfeed", handleAddFeed)
	cmdsPtr.register("feeds", handleFeeds)

	if err != nil {
		log.Fatalf("Error registering login\n")
	}

	if len(os.Args) < 2 {
		log.Fatalf("Gator needs a command name\n")
	}

	var cmd command
	cmd.name = os.Args[1]
	cmd.args = os.Args[2:]
	err = cmds.run(&s, cmd)

	if err != nil {
		log.Fatalf("Error running command: %s\n", err)
	}
}
