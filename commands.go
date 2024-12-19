package main

import (
	"context"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"time"

	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	mapOfCmds map[string]func(*state, command) error
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the login handler expects a single argument: username")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err != nil {
		return err
	}

	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("The user %s has been set\n", cmd.args[0])
	return nil

}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the register handler expects a single argument: username")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err == nil {
		return fmt.Errorf("user name %s is already registered. Try another username", cmd.args[0])
	}

	params := database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.args[0]}

	user, err := s.db.CreateUser(context.Background(), params)

	if err != nil {
		return err
	}

	fmt.Printf("User %s created at %v with ID: %v\n", user.Name, user.CreatedAt, user.ID)

	err = handlerLogin(s, cmd)

	if err != nil {
		return err
	}

	return nil
}

func handleAgg(s *state, cmd command) error {
	rssfeed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", rssfeed)
	return nil
}

func handleFeeds(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("feeds command takes no args")
	}

	feeds, err := s.db.GetFeed(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		userName, err := s.db.GetNameById(context.Background(), feed.UserID)

		if err != nil {
			return err
		}

		fmt.Printf("* %s\t%s\t%s\n", feed.Name, feed.Url, userName)
	}

	return nil
}

func handleAddFeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("addfeed takes two args: name and url")
	}

	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)

	if err != nil {
		return err
	}

	var feedParams database.CreateFeedParams
	feedParams.ID = uuid.New()
	feedParams.Name = cmd.args[0]
	feedParams.Url = cmd.args[1]
	feedParams.UserID = user.ID
	feed, err := s.db.CreateFeed(context.Background(), feedParams)

	if err != nil {
		return err
	}

	fmt.Printf("%s created a feed with name: %s, url: %s\n", user.Name, feed.Name, feed.Url)
	return nil
}

func handleUsers(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("users command takes no args")
	}

	users, err := s.db.GetUsers(context.Background())

	if err != nil {
		return err
	}

	for _, user := range users {
		fmt.Printf("* %s", user)

		if s.cfg.CurrentUserName == user {
			fmt.Printf(" (current)")
		}
		fmt.Printf("\n")
	}

	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("reset command takes no args")
	}

	err := s.db.DeleteUsers(context.Background())

	if err != nil {
		return err
	}

	fmt.Println("Deleted users table.")
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.mapOfCmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	return c.mapOfCmds[cmd.name](s, cmd)
}
