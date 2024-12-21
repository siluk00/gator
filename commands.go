package main

import (
	"context"
	"database/sql"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)

		if err != nil {
			return err
		}

		err = handler(s, cmd, user)

		if err != nil {
			return err
		}

		return nil
	}
}

func handleAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("need one arg: time between requisitions")
	}

	duration, err := time.ParseDuration(cmd.args[0])

	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s\n", cmd.args[0])

	done := make(chan struct{})
	go func() {
		time.Sleep(duration * 10)
		done <- struct{}{}
	}()

	ticker := time.NewTicker(duration)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			scrapeFeeds(s)
		case <-done:
			fmt.Println("Stop feed colletion")
			return nil
		}
	}
}

func handleUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("unfollow need url as args")
	}

	if _, err := url.ParseRequestURI(cmd.args[0]); err != nil {
		return err
	}

	var params database.DeleteFeedFollowParams
	params.Name = user.Name
	params.Url = cmd.args[0]

	err := s.db.DeleteFeedFollow(context.Background(), params)

	if err != nil {
		return err
	}

	fmt.Printf("%s user unfollowed %s succesfully\n", user.Name, cmd.args[0])

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
		userName, err := s.db.GetUserNameById(context.Background(), feed.UserID)

		if err != nil {
			return err
		}

		fmt.Printf("* %s\t%s\t%s\n", feed.Name, feed.Url, userName)
	}

	return nil
}

func handleAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed takes two args: name and url")
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

	cmd.args[0] = cmd.args[1]
	cmd.args = cmd.args[0:1]
	err = handleFollow(s, cmd, user)
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

func handleFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("expected no args")
	}

	feedFollows, err := s.db.GetFeedFollowsForUser(context.Background(), user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("* %s:\n", user.Name)
	for _, feedFollow := range feedFollows {
		fmt.Printf(" * %s\n", feedFollow.FeedName)
	}

	return nil
}

func handleBrowse(s *state, cmd command, user database.User) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf("expecting maximun of 1 arg: limit")
	}

	var limit int32
	if len(cmd.args) == 0 {
		limit = 2
	} else {
		limitInt, err := strconv.Atoi(cmd.args[0])

		if err != nil {
			return err
		}

		limit = int32(limitInt)
	}

	var params database.GetPostsForUserParams
	params.Limit = limit
	params.UserID = user.ID

	posts, err := s.db.GetPostsForUser(context.Background(), params)

	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Printf("* %s\n", post.Url)
		if post.Title.Valid {
			fmt.Printf("\t- title: %s\n", post.Title.String)
		}
		if post.Description.Valid {
			fmt.Printf("\t- description: %s\n", post.Description.String)
		}
		if post.PublishedAt.Valid {
			fmt.Printf("\t- published at: %s\n", post.PublishedAt.String)
		}
	}

	return nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())

	if err != nil {
		return err
	}

	err = s.db.MarkfeedFetched(context.Background(), feed.ID)

	if err != nil {
		return err
	}

	rssfeed, err := fetchFeed(context.Background(), feed.Url)

	if err != nil {
		return err
	}

	for _, rssitem := range rssfeed.Channel.Item {

		var title sql.NullString
		var description sql.NullString
		var publishedAt sql.NullString

		if rssitem.Title == "" {
			title.Valid = false
		} else {
			title.String = rssitem.Title
			title.Valid = true
		}

		if rssitem.Description == "" {
			description.Valid = false
		} else {
			description.String = rssitem.Description
			description.Valid = true
		}

		if rssitem.PubDate == "" {
			publishedAt.Valid = false
		} else {
			publishedAt.String = rssitem.PubDate
			publishedAt.Valid = true
		}

		var params database.CreatePostParams
		params.ID = uuid.New()
		params.CreatedAt = time.Now()
		params.Title = title
		params.Url = rssitem.Link
		params.Description = description
		params.PublishedAt = publishedAt
		params.FeedID = feed.ID

		_, err := s.db.CreatePost(context.Background(), params)

		if err != nil {
			if pqerr, ok := err.(*pq.Error); ok {
				if pqerr.Code != "23505" {
					return err
				}
			} else {
				return err
			}
		}

		fmt.Printf("* %s\n", rssitem.Title)
	}

	return nil
}

func handleFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expecting only 1 url argument")
	}

	if _, err := url.ParseRequestURI(cmd.args[0]); err != nil {
		return fmt.Errorf("not url: %s", err)
	}

	feed, err := s.db.GetFeedsByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	var params database.CreateFeedFollowParams
	params.CreatedAt = time.Now()
	params.ID = uuid.New()
	params.FeedID = feed.ID
	params.UserID = user.ID

	feedFollows, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}

	fmt.Printf("%s now follows %s\n", feedFollows.UserName, feedFollows.FeedName)
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

	err = s.db.DeleteFeeds(context.Background())
	if err != nil {
		return err
	}

	fmt.Println("Deleted feeds table.")

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.mapOfCmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	return c.mapOfCmds[cmd.name](s, cmd)
}
