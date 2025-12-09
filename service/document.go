package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/repository/pg"
)

const (
	// ExpirationWarningDays is the number of days before expiration to show yellow status
	ExpirationWarningDays = 30
)

type DocumentService interface {
	CreateDocument(ctx context.Context, req entity.CreateDocumentRequest, companyID int, fileName string, fileData []byte) (string, error)
	GetDocumentByID(ctx context.Context, id, requesterCompanyID int) (*entity.Document, error)
	GetDocumentsByCompanyID(ctx context.Context, companyID int) ([]entity.Document, error)
	VerifyDocument(ctx context.Context, hash string, requesterCompanyID, userID int) (*entity.Document, entity.DocumentStatus, string, error)
	GetHistory(ctx context.Context, userID int) ([]entity.VerificationHistory, error)
}

type documentService struct {
	documentRepo pg.DocumentRepository
	historyRepo  pg.HistoryRepository
}

func NewDocumentService(documentRepo pg.DocumentRepository, historyRepo pg.HistoryRepository) DocumentService {
	return &documentService{
		documentRepo: documentRepo,
		historyRepo:  historyRepo,
	}
}

// CreateDocument creates a new document and returns its hash
func (s *documentService) CreateDocument(ctx context.Context, req entity.CreateDocumentRequest, companyID int, fileName string, fileData []byte) (string, error) {
	doc := &entity.Document{
		CompanyID:      companyID,
		Type:           req.Type,
		Name:           req.Name,
		Summary:        req.Summary,
		ExpirationDate: req.ExpirationDate,
		FileName:       fileName,
		FileData:       fileData,
	}

	err := s.documentRepo.CreateDocument(ctx, doc)
	if err != nil {
		slog.Error("error creating document", "err", err)
		return "", errs.InternalError("error creating document", err)
	}

	// Create hash payload (id, company_id, type, name - no summary and expiration_date)
	payload := entity.DocumentHashPayload{
		ID:        doc.ID,
		CompanyID: doc.CompanyID,
		Type:      doc.Type,
		Name:      doc.Name,
	}

	// Encode payload to JSON and then to base64
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("error marshaling hash payload", "err", err)
		return "", errs.InternalError("error creating document hash", err)
	}

	hash := base64.URLEncoding.EncodeToString(payloadBytes)
	return hash, nil
}

// GetDocumentByID returns a document by its ID (only if requester belongs to the same company)
func (s *documentService) GetDocumentByID(ctx context.Context, id, requesterCompanyID int) (*entity.Document, error) {
	doc, err := s.documentRepo.GetDocumentByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errs.NotFoundError("document", err)
		}
		slog.Error("error getting document", "err", err)
		return nil, errs.InternalError("error getting document", err)
	}

	// Check if requester belongs to the same company as the document
	if doc.CompanyID != requesterCompanyID {
		return nil, errs.UnauthorizedError("access denied: you can only access documents from your own company", nil)
	}

	return &doc, nil
}

// GetDocumentsByCompanyID returns all documents for a company
func (s *documentService) GetDocumentsByCompanyID(ctx context.Context, companyID int) ([]entity.Document, error) {
	docs, err := s.documentRepo.GetDocumentsByCompanyID(ctx, companyID)
	if err != nil {
		slog.Error("error getting documents by company", "err", err)
		return nil, errs.InternalError("error getting documents", err)
	}
	return docs, nil
}

// VerifyDocument verifies a document by its hash and returns the full document with status
func (s *documentService) VerifyDocument(ctx context.Context, hash string, requesterCompanyID, userID int) (*entity.Document, entity.DocumentStatus, string, error) {
	// Decode hash
	payloadBytes, err := base64.URLEncoding.DecodeString(hash)
	if err != nil {
		slog.Error("error decoding hash", "err", err)
		return nil, entity.DocumentStatusRed, "Invalid document hash", errs.BadRequestError("invalid document hash", err)
	}

	var payload entity.DocumentHashPayload
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		slog.Error("error unmarshaling hash payload", "err", err)
		return nil, entity.DocumentStatusRed, "Invalid document hash format", errs.BadRequestError("invalid document hash format", err)
	}

	// Check if requester belongs to the same company as the document
	if payload.CompanyID != requesterCompanyID {
		return nil, entity.DocumentStatusRed, "Access denied: you can only verify documents from your own company", nil
	}

	// Fetch document from database verifying id, company_id, type and name match
	doc, err := s.documentRepo.GetDocumentByID(ctx, payload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.DocumentStatusRed, "Document not found", nil
		}
		slog.Error("error getting document", "err", err)
		return nil, entity.DocumentStatusRed, "Error verifying document", errs.InternalError("error verifying document", err)
	}

	// Determine status based on expiration date
	now := time.Now()
	status, message := s.getDocumentStatus(doc.ExpirationDate, now)

	// Increment scan count
	if err := s.documentRepo.IncrementScanCount(ctx, doc.ID); err != nil {
		slog.Error("error incrementing scan count", "err", err)
		// Don't fail the verification, just log the error
	}
	doc.ScanCount++ // Update local copy for response

	// Record verification history
	history := &entity.VerificationHistory{
		UserID:     userID,
		DocumentID: doc.ID,
		Status:     status,
		Message:    message,
	}
	if err := s.historyRepo.CreateHistory(ctx, history); err != nil {
		slog.Error("error creating verification history", "err", err)
		// Don't fail the verification, just log the error
	}

	return &doc, status, message, nil
}

// GetHistory returns verification history for a user
func (s *documentService) GetHistory(ctx context.Context, userID int) ([]entity.VerificationHistory, error) {
	history, err := s.historyRepo.GetHistoryByUserID(ctx, userID)
	if err != nil {
		slog.Error("error getting history", "err", err)
		return nil, errs.InternalError("error getting history", err)
	}
	return history, nil
}

// getDocumentStatus determines the status and message based on expiration date
func (s *documentService) getDocumentStatus(expirationDate, now time.Time) (entity.DocumentStatus, string) {
	if expirationDate.Before(now) {
		return entity.DocumentStatusRed, "Document has expired"
	}

	daysUntilExpiration := int(expirationDate.Sub(now).Hours() / 24)
	if daysUntilExpiration <= ExpirationWarningDays {
		return entity.DocumentStatusYellow, "Document will expire soon"
	}

	return entity.DocumentStatusGreen, "Document is valid"
}
