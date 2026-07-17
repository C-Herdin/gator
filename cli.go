package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/C-Herdin/Gator/internal/database"
	"github.com/google/uuid"
)

var insufficient_args_error error = errors.New("insufficient arguments")

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.handlers[cmd.name]
	if !ok {
		return errors.New("invalid command")
	}
	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func (c *commands) registerAll() {
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", handlerAddFeed)
	c.register("feeds", handlerFeeds)
	c.register("follow", handlerFollow)
	c.register("following", handlerFollowing)
	c.register("unfollow", handlerUnfollow)
	c.register("browse", handlerBrowse)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return insufficient_args_error
	}
	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("User %v doesn't exists\n", cmd.args[0])
		} else {
			return err
		}
	}
	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Printf("User %v has been set\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return insufficient_args_error
	}
	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err == nil {
		return fmt.Errorf("User %v already exists\n", cmd.args[0])
	} else if err != sql.ErrNoRows {
		return err
	}
	currentTime := getCurrentTime()
	user, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Name:      cmd.args[0],
		},
	)
	if err != nil {
		return err
	}
	if err := s.cfg.SetUser(user.Name); err != nil {
		return err
	}
	fmt.Println("User", user.Name, "was created successfully")
	return nil
}

func handlerReset(s *state, cmd command) error {
	return s.db.ResetUsers(context.Background())
}

func handlerUsers(s *state, cmd command) error {
	names, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Println("No users have been logged")
		return nil
	}
	for _, n := range names {
		if s.cfg.CurrentUserName == n {
			fmt.Println("*", n, "(current)")
		} else {
			fmt.Println("*", n)
		}
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return insufficient_args_error
	}
	inputTimeBetweenRequests := cmd.args[0]
	timeBetweenRequests, err := time.ParseDuration(inputTimeBetweenRequests)
	if err != nil {
		return err
	}
	fmt.Println("Collecting feeds every", inputTimeBetweenRequests)
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return insufficient_args_error
	}
	name := cmd.args[0]
	url := cmd.args[1]
	feed, err := s.createFeed(context.Background(), name, url)
	if err != nil {
		return err
	}
	if _, err := s.createFeedFollow(context.Background(), feed.ID); err != nil {
		return err
	}
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds have been logged")
		return nil
	}
	for _, f := range feeds {
		fmt.Printf("Name: %v,\n    Url: %v,\n    Created by: %v\n", f.Name, f.Url, f.UserName)
	}
	return nil
}

func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return insufficient_args_error
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeedFromUrl(context.Background(), url)
	if err != nil {
		return err
	}
	newFeedFollow, err := s.createFeedFollow(context.Background(), feed.ID)
	if err != nil {
		return err
	}
	fmt.Printf("User %v is now following %v.", newFeedFollow.UserName, newFeedFollow.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command) error {
	feedFollows, err := s.db.GetFeedFollowsForUser(
		context.Background(),
		s.cfg.CurrentUserName,
	)
	if err != nil {
		return err
	}
	fmt.Printf("User %v currently follows:\n", s.cfg.CurrentUserName)
	for _, fu := range feedFollows {
		fmt.Println(" -", fu.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return insufficient_args_error
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeedFromUrl(context.Background(), url)
	if err != nil {
		return err
	}
	userID, err := s.getCurrentUserID(context.Background())
	if err != nil {
		return err
	}
	if err := s.db.DeleteFeedFollow(
		context.Background(),
		database.DeleteFeedFollowParams{
			UserID: userID,
			FeedID: feed.ID,
		},
	); err != nil {
		return err
	}
	return nil
}

func handlerBrowse(s *state, cmd command) error {
	var limit int = 2
	var err error
	if len(cmd.args) != 0 {
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return errors.New("Failed to parse optional argument. Should be an integer.")
		}
	}
	posts, err := s.db.GetUserPosts(context.Background(), int32(limit))
	if err != nil {
		return err
	}
	for _, post := range posts {
		fmt.Printf(
			"= Title: %v\n-   Published at: %v\n-   Description: %v\n",
			post.Title, post.PublishedAt.Time, post.Description.String,
		)
	}
	return nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}
	if err := s.db.MarkFeedFetched(
		context.Background(),
		database.MarkFeedFetchedParams{
			LastFetchedAt: getCurrentTime(),
			ID:            feed.ID,
		},
	); err != nil {
		return err
	}
	rss, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}
	for _, item := range rss.Channel.Item {
		currentTime := getCurrentTime()
		description := sql.NullString{}
		if item.Description != "" {
			description.String = item.Description
			description.Valid = true
		}
		publicationDate := sql.NullTime{}
		if pubDate, err := time.Parse(time.Layout, item.PubDate); err == nil {
			publicationDate.Time = pubDate
			publicationDate.Valid = true
		}

		if _, err := s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   currentTime,
				UpdatedAt:   currentTime,
				Title:       item.Title,
				Url:         item.Link,
				Description: description,
				PublishedAt: publicationDate,
				FeedID:      feed.ID,
			},
		); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}
