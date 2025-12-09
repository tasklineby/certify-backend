package pg

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/tasklineby/certify-backend/entity"
)

type DocumentRepository interface {
	CreateDocument(ctx context.Context, doc *entity.Document) error
	GetDocumentByID(ctx context.Context, id int) (entity.Document, error)
	IncrementScanCount(ctx context.Context, id int) error
}

type documentRepository struct {
	db *sqlx.DB
}

func NewDocumentRepository(db *sqlx.DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) CreateDocument(ctx context.Context, doc *entity.Document) error {
	query := `INSERT INTO documents (company_id, type, name, summary, expiration_date) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := r.db.QueryRowContext(ctx, query,
		doc.CompanyID, doc.Type, doc.Name, doc.Summary, doc.ExpirationDate).
		Scan(&doc.ID)
	if err != nil {
		slog.Error("error creating document", "err", err, "name", doc.Name)
		return err
	}
	return nil
}

func (r *documentRepository) GetDocumentByID(ctx context.Context, id int) (entity.Document, error) {
	query := `SELECT id, company_id, type, name, summary, expiration_date, scan_count 
	          FROM documents WHERE id = $1`
	var doc entity.Document
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.CompanyID, &doc.Type, &doc.Name, &doc.Summary, &doc.ExpirationDate, &doc.ScanCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Document{}, err
		}
		slog.Error("error getting document by id", "err", err, "document_id", id)
		return entity.Document{}, err
	}
	return doc, nil
}

func (r *documentRepository) IncrementScanCount(ctx context.Context, id int) error {
	query := `UPDATE documents SET scan_count = scan_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		slog.Error("error incrementing scan count", "err", err, "document_id", id)
		return err
	}
	return nil
}
