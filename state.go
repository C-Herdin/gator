package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/C-Herdin/Gator/internal/config"
	"github.com/C-Herdin/Gator/internal/database"
	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

func (s *state) getCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	user, err := s.db.GetUser(ctx, s.cfg.CurrentUserName)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not get user because of %v", err)
	}
	return user.ID, nil
}

func (s *state) createFeed(ctx context.Context, name, url string) (database.Feed, error) {
	currentTime := getCurrentTime()
	userID, err := s.getCurrentUserID(ctx)
	if err != nil {
		return database.Feed{}, err
	}
	return s.db.CreateFeed(
		ctx,
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Name:      name,
			Url:       url,
			UserID:    userID,
		},
	)
}

func (s *state) createFeedFollow(ctx context.Context, feedID uuid.UUID) (database.CreateFeedFollowRow, error) {
	currentTime := getCurrentTime()
	userID, err := s.getCurrentUserID(ctx)
	if err != nil {
		return database.CreateFeedFollowRow{}, err
	}
	return s.db.CreateFeedFollow(
		ctx,
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			UserID:    userID,
			FeedID:    feedID,
		},
	)
}

func getCurrentTime() sql.NullTime {
	return sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
}
