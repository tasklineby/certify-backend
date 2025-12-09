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
	CreateDocument(ctx context.Context, req entity.CreateDocumentRequest, companyID int) (string, error)
	VerifyDocument(ctx context.Context, hash string, requesterCompanyID int) (*entity.Document, entity.DocumentStatus, string, error)
}

type documentService struct {
	documentRepo pg.DocumentRepository
}

func NewDocumentService(documentRepo pg.DocumentRepository) DocumentService {
	return &documentService{
		documentRepo: documentRepo,
	}
}

// CreateDocument creates a new document and returns its hash
func (s *documentService) CreateDocument(ctx context.Context, req entity.CreateDocumentRequest, companyID int) (string, error) {
	doc := &entity.Document{
		CompanyID:      companyID,
		Type:           req.Type,
		Name:           req.Name,
		Summary:        req.Summary,
		ExpirationDate: req.ExpirationDate,
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

// VerifyDocument verifies a document by its hash and returns the full document with status
func (s *documentService) VerifyDocument(ctx context.Context, hash string, requesterCompanyID int) (*entity.Document, entity.DocumentStatus, string, error) {
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

	return &doc, status, message, nil
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
