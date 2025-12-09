package rdb

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
)

type TokenRepository interface {
	SetRefreshToken(ctx context.Context, token entity.RefreshToken) error
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	BlacklistAccessToken(ctx context.Context, tokenHash string, remaining time.Duration) error
	IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	FetchUserFromRefreshToken(ctx context.Context, tokenHash string) (entity.TokenPayload, error)
}

type tokenRepository struct {
	rdb             *redis.Client
	blacklistPrefix string
	refreshPrefix   string
}

func NewTokenRepository(rdb *redis.Client) TokenRepository {
	return &tokenRepository{
		rdb:             rdb,
		blacklistPrefix: "blacklist:",
		refreshPrefix:   "refresh:",
	}
}

func (r *tokenRepository) SetRefreshToken(ctx context.Context, token entity.RefreshToken) error {
	status := r.rdb.Set(ctx, r.refreshPrefix+token.Token, token.UserID, token.ExpiresIn)
	err := status.Err()
	if err != nil {
		slog.Error("error setting refresh token to database", "err", err)
		return errs.InternalError("error setting refresh token to database", err)
	}
	return nil
}

func (r *tokenRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	if err := r.rdb.Del(ctx, r.refreshPrefix+tokenHash).Err(); err != nil {
		slog.Error("error deleting refresh token from db", "err", err)
		return errs.InternalError("error deleting refresh token from database", err)
	}
	return nil
}

func (r *tokenRepository) BlacklistAccessToken(ctx context.Context, tokenHash string, remaining time.Duration) error {
	if err := r.rdb.Set(ctx, r.blacklistPrefix+tokenHash, "token", remaining).Err(); err != nil {
		slog.Error("error blacklisting access token", "err", err)
		return errs.InternalError("error blacklisting access token", err)
	}
	return nil
}

func (r *tokenRepository) IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	exists, err := r.rdb.Exists(ctx, r.blacklistPrefix+tokenHash).Result()
	if err != nil {
		slog.Error("error validating token", "err", err)
		return false, errs.InternalError("error validating token", err)
	}
	return exists > 0, nil
}

func (r *tokenRepository) FetchUserFromRefreshToken(ctx context.Context, tokenHash string) (entity.TokenPayload, error) {
	jsonData, err := r.rdb.Get(ctx, r.refreshPrefix+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		slog.Error("refresh token not found", "err", err)
		return entity.TokenPayload{}, errs.NotFoundError("refresh token", err)
	}
	if err != nil {
		slog.Error("error fetching user id from db", "err", err)
		return entity.TokenPayload{}, errs.InternalError("error fetching user id from db", err)
	}
	var payload entity.TokenPayload
	err = json.Unmarshal([]byte(jsonData), &payload)
	if err != nil {
		slog.Error("error unmarshaling token payload", "err", err)
		return entity.TokenPayload{}, errs.InternalError("error unmarshaling token payload", err)
	}
	return payload, nil
}
