package entity

import (
	"time"
)

// User represents a user entity
type User struct {
	ID        int       `db:"id" json:"id"`
	Role      string    `db:"role" json:"role"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"-"`
	CompanyID int       `db:"company_id" json:"company_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Company represents a company entity
type Company struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// TokenPayload represents the payload in JWT tokens
type TokenPayload struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	CompanyID string `json:"company_id"`
}

// RefreshToken represents a refresh token stored in Redis
type RefreshToken struct {
	UserID    string        `json:"user_id"`
	Token     string        `json:"token"`
	ExpiresIn time.Duration `json:"expires_in"`
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// CreateCompanyRequest represents request to create company with admin
type CreateCompanyRequest struct {
	CompanyName string    `json:"company_name" binding:"required"`
	Admin       AdminUser `json:"admin" binding:"required"`
}

// AdminUser represents admin user data
type AdminUser struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
}

// RegisterEmployeeRequest represents request to register employee
type RegisterEmployeeRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	CompanyID int    `json:"company_id" binding:"required"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateUserRequest represents request to update user profile
type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email" binding:"omitempty,email"`
}
