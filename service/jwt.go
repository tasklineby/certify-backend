package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
)

func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func secureRandomBase64() (string, error) {
	buf := make([]byte, 64)
	if _, err := rand.Read(buf); err != nil {
		return "", errs.InternalError("error creating secure random base 64", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type AccessToken struct {
	entity.TokenPayload
	jwt.RegisteredClaims
}

type JwtService interface {
	GenerateAccessToken(ctx context.Context, payload entity.TokenPayload) (string, error)
	GenerateRefreshToken(ctx context.Context, payload entity.TokenPayload) (entity.RefreshToken, error)
	ParseAccessToken(ctx context.Context, token string) (entity.TokenPayload, time.Time, error)
}

type jwtService struct {
	accessTokenSecret string
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
}

func NewJwtService(accessTokenSecret string, accessTokenTTL, refreshTokenTTL time.Duration) JwtService {
	return &jwtService{
		accessTokenSecret: accessTokenSecret,
		accessTokenTTL:    accessTokenTTL,
		refreshTokenTTL:   refreshTokenTTL,
	}
}

func (s *jwtService) GenerateAccessToken(ctx context.Context, payload entity.TokenPayload) (string, error) {
	tokenClaims := AccessToken{
		TokenPayload: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	return token.SignedString([]byte(s.accessTokenSecret))
}

func (s *jwtService) GenerateRefreshToken(ctx context.Context, payload entity.TokenPayload) (entity.RefreshToken, error) {
	str, err := secureRandomBase64()
	if err != nil {
		slog.Error("error generating refresh token", "err", err)
		return entity.RefreshToken{}, errs.InternalError("error generating refresh token", err)
	}
	hash := SHA256Hex(str)
	token := entity.RefreshToken{
		UserID:    payload.UserID,
		Token:     hash,
		ExpiresIn: s.refreshTokenTTL,
	}
	return token, nil
}

func (s *jwtService) ParseAccessToken(ctx context.Context, token string) (entity.TokenPayload, time.Time, error) {
	var accessToken AccessToken
	parsedToken, err := jwt.ParseWithClaims(token, &accessToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.accessTokenSecret), nil
	})
	if err != nil {
		slog.Error("error parsing access token", "err", err)
		return entity.TokenPayload{}, time.Time{}, errs.InternalError("error parsing access token", err)
	}
	if !parsedToken.Valid {
		return entity.TokenPayload{}, time.Time{}, errs.UnauthorizedError("invalid access token", err)
	}
	expTime := time.Time{}
	if accessToken.ExpiresAt != nil {
		expTime = accessToken.ExpiresAt.Time
	}
	return accessToken.TokenPayload, expTime, nil
}
