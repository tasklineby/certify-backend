package entity

import (
	"time"
)

// User represents a user entity
// @Description User entity with profile information
type User struct {
	ID        int       `db:"id" json:"id" example:"1"`
	Role      string    `db:"role" json:"role" example:"employee"`
	FirstName string    `db:"first_name" json:"first_name" example:"John"`
	LastName  string    `db:"last_name" json:"last_name" example:"Doe"`
	Email     string    `db:"email" json:"email" example:"user@example.com"`
	Password  string    `db:"password" json:"-"`
	CompanyID int       `db:"company_id" json:"company_id" example:"1"`
	CreatedAt time.Time `db:"created_at" json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// Company represents a company entity
// @Description Company entity
type Company struct {
	ID   int    `db:"id" json:"id" example:"1"`
	Name string `db:"name" json:"name" example:"Acme Corp"`
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
// @Description Token pair response for authentication
type TokenPair struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"abc123def456..."`
}

// CreateCompanyRequest represents request to create company with admin
// @Description Request to create a company and register its admin
type CreateCompanyRequest struct {
	CompanyName string    `json:"company_name" binding:"required" example:"Acme Corp"`
	Admin       AdminUser `json:"admin" binding:"required"`
}

// AdminUser represents admin user data
// @Description Admin user registration data
type AdminUser struct {
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
	Email     string `json:"email" binding:"required,email" example:"admin@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"password123"`
}

// RegisterEmployeeRequest represents request to register employee
// @Description Request to register a new employee user
type RegisterEmployeeRequest struct {
	FirstName string `json:"first_name" binding:"required" example:"Jane"`
	LastName  string `json:"last_name" binding:"required" example:"Smith"`
	Email     string `json:"email" binding:"required,email" example:"employee@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"password123"`
	CompanyID int    `json:"company_id" binding:"required" example:"1"`
}

// LoginRequest represents login request
// @Description Login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// RefreshRequest represents refresh token request
// @Description Refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"abc123def456..."`
}

// UpdateUserRequest represents request to update user profile
// @Description Request to update user profile (all fields optional)
type UpdateUserRequest struct {
	FirstName *string `json:"first_name" example:"John"`
	LastName  *string `json:"last_name" example:"Doe"`
	Email     *string `json:"email" binding:"omitempty,email" example:"newemail@example.com"`
}
