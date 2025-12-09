package pg

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/tasklineby/certify-backend/entity"
)

type HistoryRepository interface {
	CreateHistory(ctx context.Context, history *entity.VerificationHistory) error
	GetHistoryByUserID(ctx context.Context, userID int) ([]entity.VerificationHistory, error)
}

type historyRepository struct {
	db *sqlx.DB
}

func NewHistoryRepository(db *sqlx.DB) HistoryRepository {
	return &historyRepository{db: db}
}

func (r *historyRepository) CreateHistory(ctx context.Context, history *entity.VerificationHistory) error {
	query := `INSERT INTO verification_history (user_id, document_id, status, message) 
	          VALUES ($1, $2, $3, $4) RETURNING id, scanned_at`
	err := r.db.QueryRowContext(ctx, query,
		history.UserID, history.DocumentID, history.Status, history.Message).
		Scan(&history.ID, &history.ScannedAt)
	if err != nil {
		slog.Error("error creating verification history", "err", err, "user_id", history.UserID)
		return err
	}
	return nil
}

func (r *historyRepository) GetHistoryByUserID(ctx context.Context, userID int) ([]entity.VerificationHistory, error) {
	query := `SELECT id, user_id, document_id, status, message, scanned_at 
	          FROM verification_history WHERE user_id = $1 ORDER BY scanned_at DESC`
	var history []entity.VerificationHistory
	err := r.db.SelectContext(ctx, &history, query, userID)
	if err != nil {
		slog.Error("error getting history by user id", "err", err, "user_id", userID)
		return nil, err
	}
	return history, nil
}
