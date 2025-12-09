package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/repository/rdb"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (entity.TokenPair, error)
	Register(ctx context.Context, req entity.RegisterEmployeeRequest) (entity.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (entity.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ParseToken(ctx context.Context, token string) (entity.TokenPayload, error)
}

type authService struct {
	userService UserService
	tokenRepo   rdb.TokenRepository
	jwtService  JwtService
}

func NewAuthService(userService UserService, tokenRepo rdb.TokenRepository, jwtService JwtService) AuthService {
	return &authService{
		userService: userService,
		tokenRepo:   tokenRepo,
		jwtService:  jwtService,
	}
}

func (s *authService) Login(ctx context.Context, email, password string) (entity.TokenPair, error) {
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		slog.Error("error getting user by email", "err", err)
		return entity.TokenPair{}, errs.UnauthorizedError("invalid credentials", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return entity.TokenPair{}, errs.UnauthorizedError("invalid credentials", err)
	}

	tokenPayload := entity.TokenPayload{
		UserID:    strconv.Itoa(user.ID),
		Role:      user.Role,
		CompanyID: strconv.Itoa(user.CompanyID),
	}

	accessToken, err := s.jwtService.GenerateAccessToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating access token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating access token", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating refresh token", err)
	}

	err = s.tokenRepo.SetRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error setting refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error setting refresh token", err)
	}

	return entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *authService) Register(ctx context.Context, req entity.RegisterEmployeeRequest) (entity.TokenPair, error) {
	return s.userService.RegisterEmployee(ctx, req, s.jwtService, s.tokenRepo)
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (entity.TokenPair, error) {
	payload, err := s.tokenRepo.FetchUserFromRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error fetching user from refresh token", "err", err)
		return entity.TokenPair{}, err
	}

	err = s.tokenRepo.DeleteRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error deleting refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error deleting refresh token", err)
	}

	accessToken, err := s.jwtService.GenerateAccessToken(ctx, payload)
	if err != nil {
		slog.Error("error generating access token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating access token", err)
	}

	newRefreshToken, err := s.jwtService.GenerateRefreshToken(ctx, payload)
	if err != nil {
		slog.Error("error generating refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating refresh token", err)
	}

	err = s.tokenRepo.SetRefreshToken(ctx, newRefreshToken)
	if err != nil {
		slog.Error("error setting refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error setting refresh token", err)
	}

	return entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
	}, nil
}

func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	err := s.tokenRepo.DeleteRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error deleting refresh token", "err", err)
		return err
	}

	_, expTime, err := s.jwtService.ParseAccessToken(ctx, accessToken)
	if err != nil {
		slog.Error("error parsing access token", "err", err)
		return err
	}

	if remaining := time.Until(expTime); remaining > 0 {
		tokenHash := SHA256Hex(accessToken)
		err = s.tokenRepo.BlacklistAccessToken(ctx, tokenHash, remaining)
		if err != nil {
			slog.Error("error blacklisting access token", "err", err)
			return err
		}
	}

	return nil
}

func (s *authService) ParseToken(ctx context.Context, token string) (entity.TokenPayload, error) {
	tokenHash := SHA256Hex(token)
	isBlacklisted, err := s.tokenRepo.IsAccessTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		slog.Error("error verifying access token", "err", err)
		return entity.TokenPayload{}, errs.UnauthorizedError("invalid token", err)
	}
	if isBlacklisted {
		return entity.TokenPayload{}, errs.UnauthorizedError("token has been revoked", nil)
	}

	tokenPayload, _, err := s.jwtService.ParseAccessToken(ctx, token)
	if err != nil {
		slog.Error("error parsing access token", "err", err)
		return entity.TokenPayload{}, errs.UnauthorizedError("invalid token", err)
	}

	return tokenPayload, nil
}
