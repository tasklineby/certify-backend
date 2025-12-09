package pg

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/tasklineby/certify-backend/entity"
)

type CompanyRepository interface {
	CreateCompany(ctx context.Context, company *entity.Company) error
	GetCompanyByID(ctx context.Context, id int) (entity.Company, error)
	UpdateCompany(ctx context.Context, id int, name string) error
	DeleteCompany(ctx context.Context, id int) error
}

type companyRepository struct {
	db *sqlx.DB
}

func NewCompanyRepository(db *sqlx.DB) CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) CreateCompany(ctx context.Context, company *entity.Company) error {
	query := `INSERT INTO companies (name) VALUES ($1) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, company.Name).Scan(&company.ID)
	if err != nil {
		slog.Error("error creating company", "err", err, "name", company.Name)
		return err
	}
	return nil
}

func (r *companyRepository) GetCompanyByID(ctx context.Context, id int) (entity.Company, error) {
	query := `SELECT id, name FROM companies WHERE id = $1`
	var company entity.Company
	err := r.db.QueryRowContext(ctx, query, id).Scan(&company.ID, &company.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Company{}, err
		}
		slog.Error("error getting company by id", "err", err, "company_id", id)
		return entity.Company{}, err
	}
	return company, nil
}

func (r *companyRepository) UpdateCompany(ctx context.Context, id int, name string) error {
	query := `UPDATE companies SET name = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, name, id)
	if err != nil {
		slog.Error("error updating company", "err", err, "company_id", id, "name", name)
		return err
	}
	return nil
}

func (r *companyRepository) DeleteCompany(ctx context.Context, id int) error {
	query := `DELETE FROM companies WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		slog.Error("error deleting company", "err", err, "company_id", id)
		return err
	}
	return nil
}
