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

// Document represents a document entity
// @Description Document entity with type, name, summary and expiration date
type Document struct {
	ID             int       `db:"id" json:"id" example:"1"`
	CompanyID      int       `db:"company_id" json:"company_id" example:"1"`
	Type           string    `db:"type" json:"type" example:"agreement"`
	Name           string    `db:"name" json:"name" example:"Employment Agreement"`
	Summary        string    `db:"summary" json:"summary" example:"Standard employment agreement for full-time employees"`
	ExpirationDate time.Time `db:"expiration_date" json:"expiration_date" example:"2025-12-31T00:00:00Z"`
	ScanCount      int       `db:"scan_count" json:"scan_count" example:"42"`
	FileName       string    `db:"file_name" json:"file_name" example:"contract.pdf"`
	FileData       []byte    `db:"file_data" json:"-"`
}

// VerificationHistory represents a document verification history entry
// @Description Record of a document verification attempt
type VerificationHistory struct {
	ID         int            `db:"id" json:"id" example:"1"`
	UserID     int            `db:"user_id" json:"user_id" example:"1"`
	DocumentID int            `db:"document_id" json:"document_id" example:"1"`
	Status     DocumentStatus `db:"status" json:"status" example:"green"`
	Message    string         `db:"message" json:"message" example:"Document is valid"`
	ScannedAt  time.Time      `db:"scanned_at" json:"scanned_at" example:"2024-01-01T12:00:00Z"`
}

// DocumentStatus represents the status of a document based on expiration
type DocumentStatus string

const (
	DocumentStatusGreen  DocumentStatus = "green"  // Document is valid
	DocumentStatusYellow DocumentStatus = "yellow" // Document will expire soon (within 30 days)
	DocumentStatusRed    DocumentStatus = "red"    // Document is expired or not found
)

// CreateDocumentRequest represents request to create a document
// @Description Request to create a new document
type CreateDocumentRequest struct {
	Type           string    `json:"type" binding:"required" example:"agreement"`
	Name           string    `json:"name" binding:"required" example:"Employment Agreement"`
	Summary        string    `json:"summary" binding:"required" example:"Standard employment agreement for full-time employees"`
	ExpirationDate time.Time `json:"expiration_date" binding:"required" example:"2025-12-31T00:00:00Z"`
}

// CreateDocumentResponse represents response after creating a document
// @Description Response containing the document hash for later retrieval
type CreateDocumentResponse struct {
	Hash string `json:"hash" example:"eyJpZCI6MSwidHlwZSI6ImFncmVlbWVudCIsIm5hbWUiOiJFbXBsb3ltZW50IEFncmVlbWVudCJ9"`
}

// VerifyDocumentResponse represents response with document details and status
// @Description Response containing full document details and verification status
type VerifyDocumentResponse struct {
	Document *Document      `json:"document"`
	Status   DocumentStatus `json:"status" example:"green"`
	Message  string         `json:"message" example:"Document is valid"`
}

// DocumentHashPayload represents the payload encoded in the document hash
type DocumentHashPayload struct {
	ID        int    `json:"id"`
	CompanyID int    `json:"company_id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
}

// DocumentAnalysisResult represents the result of document comparison analysis
// @Description Analysis result comparing uploaded document/photos with original
type DocumentAnalysisResult struct {
	Score   float64 `json:"score" example:"0.95"`
	Message string  `json:"message" example:"Documents match with 95% confidence"`
}

// CompareDocumentResponse represents the response for document comparison
// @Description Response containing document verification status, details and analysis result
type CompareDocumentResponse struct {
	Status   DocumentStatus          `json:"status" example:"green"`
	Message  string                  `json:"message" example:"Document is valid"`
	Document *Document               `json:"document"`
	Analysis *DocumentAnalysisResult `json:"analysis,omitempty"`
}
