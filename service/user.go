package service

import (
	"context"
	"database/sql"
	"log/slog"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/repository/pg"
	"github.com/tasklineby/certify-backend/repository/rdb"
	"golang.org/x/crypto/bcrypt"
)

// isUniqueConstraintError checks if the error is a PostgreSQL unique constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// Check for PostgreSQL error code 23505 (unique_violation)
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	// Also check error message as fallback
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "violates unique constraint")
}

type UserService interface {
	CreateCompanyWithAdmin(ctx context.Context, req entity.CreateCompanyRequest, jwtService JwtService, tokenRepo rdb.TokenRepository) (entity.TokenPair, error)
	RegisterEmployee(ctx context.Context, req entity.RegisterEmployeeRequest, jwtService JwtService, tokenRepo rdb.TokenRepository) (entity.TokenPair, error)
	GetUserByID(ctx context.Context, id int) (entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (entity.User, error)
	UpdateUser(ctx context.Context, id int, req entity.UpdateUserRequest, requesterRole string, requesterCompanyID int, requesterID int) error
	DeleteUser(ctx context.Context, id int, requesterRole string, requesterCompanyID int) error
	GetUsersByCompanyID(ctx context.Context, companyID int) ([]entity.User, error)
}

type userService struct {
	userRepo    pg.UserRepository
	companyRepo pg.CompanyRepository
}

func NewUserService(userRepo pg.UserRepository, companyRepo pg.CompanyRepository) UserService {
	return &userService{
		userRepo:    userRepo,
		companyRepo: companyRepo,
	}
}

func (s *userService) CreateCompanyWithAdmin(ctx context.Context, req entity.CreateCompanyRequest, jwtService JwtService, tokenRepo rdb.TokenRepository) (entity.TokenPair, error) {
	company := &entity.Company{
		Name: req.CompanyName,
	}

	err := s.companyRepo.CreateCompany(ctx, company)
	if err != nil {
		slog.Error("error creating company", "err", err)
		return entity.TokenPair{}, errs.InternalError("error creating company", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", "err", err)
		_ = s.companyRepo.DeleteCompany(ctx, company.ID)
		return entity.TokenPair{}, errs.InternalError("error hashing password", err)
	}

	// Create admin user with company_id
	adminUser := &entity.User{
		Role:      "admin",
		FirstName: req.Admin.FirstName,
		LastName:  req.Admin.LastName,
		Email:     req.Admin.Email,
		Password:  string(hashedPassword),
		CompanyID: company.ID,
	}

	err = s.userRepo.CreateUser(ctx, adminUser)
	if err != nil {
		slog.Error("error creating admin user", "err", err)
		// Cleanup: delete company if user creation fails
		_ = s.companyRepo.DeleteCompany(ctx, company.ID)
		// Check if error is due to unique constraint violation
		if isUniqueConstraintError(err) {
			return entity.TokenPair{}, errs.AlreadyExistsError("email", err)
		}
		return entity.TokenPair{}, errs.InternalError("error creating admin user", err)
	}

	// Generate tokens
	tokenPayload := entity.TokenPayload{
		UserID:    strconv.Itoa(adminUser.ID),
		Role:      adminUser.Role,
		CompanyID: strconv.Itoa(company.ID),
	}

	accessToken, err := jwtService.GenerateAccessToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating access token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating access token", err)
	}

	refreshToken, err := jwtService.GenerateRefreshToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating refresh token", err)
	}

	err = tokenRepo.SetRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error setting refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error setting refresh token", err)
	}

	return entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *userService) RegisterEmployee(ctx context.Context, req entity.RegisterEmployeeRequest, jwtService JwtService, tokenRepo rdb.TokenRepository) (entity.TokenPair, error) {
	// Verify company exists
	_, err := s.companyRepo.GetCompanyByID(ctx, req.CompanyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.TokenPair{}, errs.NotFoundError("company", err)
		}
		slog.Error("error getting company", "err", err)
		return entity.TokenPair{}, errs.InternalError("error getting company", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", "err", err)
		return entity.TokenPair{}, errs.InternalError("error hashing password", err)
	}

	user := &entity.User{
		Role:      "employee",
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CompanyID: req.CompanyID,
	}

	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		slog.Error("error creating user", "err", err)
		// Check if error is due to unique constraint violation
		if isUniqueConstraintError(err) {
			return entity.TokenPair{}, errs.AlreadyExistsError("email", err)
		}
		return entity.TokenPair{}, errs.InternalError("error creating user", err)
	}

	tokenPayload := entity.TokenPayload{
		UserID:    strconv.Itoa(user.ID),
		Role:      user.Role,
		CompanyID: strconv.Itoa(user.CompanyID),
	}

	accessToken, err := jwtService.GenerateAccessToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating access token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating access token", err)
	}

	refreshToken, err := jwtService.GenerateRefreshToken(ctx, tokenPayload)
	if err != nil {
		slog.Error("error generating refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error generating refresh token", err)
	}

	err = tokenRepo.SetRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Error("error setting refresh token", "err", err)
		return entity.TokenPair{}, errs.InternalError("error setting refresh token", err)
	}

	return entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id int) (entity.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.User{}, errs.NotFoundError("user", err)
		}
		slog.Error("error getting user", "err", err)
		return entity.User{}, errs.InternalError("error getting user", err)
	}
	return user, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.User{}, errs.NotFoundError("user", err)
		}
		slog.Error("error getting user by email", "err", err)
		return entity.User{}, errs.InternalError("error getting user by email", err)
	}
	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, id int, req entity.UpdateUserRequest, requesterRole string, requesterCompanyID int, requesterID int) error {
	// Get target user
	targetUser, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errs.NotFoundError("user", err)
		}
		slog.Error("error getting user", "err", err)
		return errs.InternalError("error getting user", err)
	}

	// If updating another user (not self), must be admin from same company
	if id != requesterID {
		if requesterRole != "admin" {
			return errs.UnauthorizedError("only admins can update other users", nil)
		}
		// Verify user belongs to same company as requester
		if targetUser.CompanyID != requesterCompanyID {
			return errs.UnauthorizedError("can only update users from the same company", nil)
		}
	} else {
		// User updating themselves - verify they belong to the same company
		if targetUser.CompanyID != requesterCompanyID {
			return errs.UnauthorizedError("user company mismatch", nil)
		}
	}

	user := &entity.User{
		FirstName: "",
		LastName:  "",
		Email:     "",
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Email != nil {
		// Check if email is already taken by another user
		existingUser, err := s.userRepo.GetUserByEmail(ctx, *req.Email)
		if err == nil && existingUser.ID != id {
			return errs.AlreadyExistsError("email", nil)
		}
		if err != nil && err != sql.ErrNoRows {
			slog.Error("error checking email", "err", err)
			return errs.InternalError("error checking email", err)
		}
		user.Email = *req.Email
	}

	err = s.userRepo.UpdateUser(ctx, id, user)
	if err != nil {
		if err == sql.ErrNoRows {
			return errs.NotFoundError("user", err)
		}
		slog.Error("error updating user", "err", err)
		return errs.InternalError("error updating user", err)
	}
	return nil
}

func (s *userService) DeleteUser(ctx context.Context, id int, requesterRole string, requesterCompanyID int) error {
	// Only admins can delete users
	if requesterRole != "admin" {
		return errs.UnauthorizedError("only admins can delete users", nil)
	}

	// Get target user to verify they belong to the same company
	targetUser, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errs.NotFoundError("user", err)
		}
		slog.Error("error getting user", "err", err)
		return errs.InternalError("error getting user", err)
	}

	// Verify user belongs to same company as requester
	if targetUser.CompanyID != requesterCompanyID {
		return errs.UnauthorizedError("can only delete users from the same company", nil)
	}

	err = s.userRepo.DeleteUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errs.NotFoundError("user", err)
		}
		slog.Error("error deleting user", "err", err)
		return errs.InternalError("error deleting user", err)
	}
	return nil
}

func (s *userService) GetUsersByCompanyID(ctx context.Context, companyID int) ([]entity.User, error) {
	// Verify company exists
	_, err := s.companyRepo.GetCompanyByID(ctx, companyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errs.NotFoundError("company", err)
		}
		slog.Error("error getting company", "err", err)
		return nil, errs.InternalError("error getting company", err)
	}

	users, err := s.userRepo.GetUsersByCompanyID(ctx, companyID)
	if err != nil {
		slog.Error("error getting users by company", "err", err)
		return nil, errs.InternalError("error getting users", err)
	}
	return users, nil
}
