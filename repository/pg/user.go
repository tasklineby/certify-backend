package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/tasklineby/certify-backend/entity"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByID(ctx context.Context, id int) (entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (entity.User, error)
	UpdateUser(ctx context.Context, id int, user *entity.User) error
	DeleteUser(ctx context.Context, id int) error
	GetUsersByCompanyID(ctx context.Context, companyID int) ([]entity.User, error)
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *entity.User) error {
	query := `INSERT INTO users (role, first_name, last_name, email, password, company_id) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query,
		user.Role, user.FirstName, user.LastName, user.Email, user.Password, user.CompanyID).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		slog.Error("error creating user", "err", err, "email", user.Email)
		return err
	}
	return nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id int) (entity.User, error) {
	query := `SELECT id, role, first_name, last_name, email, password, company_id, created_at, updated_at 
	          FROM users WHERE id = $1`
	var user entity.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Role, &user.FirstName, &user.LastName,
		&user.Email, &user.Password, &user.CompanyID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.User{}, err
		}
		slog.Error("error getting user by id", "err", err, "user_id", id)
		return entity.User{}, err
	}
	return user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	query := `SELECT id, role, first_name, last_name, email, password, company_id, created_at, updated_at 
	          FROM users WHERE email = $1`
	var user entity.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Role, &user.FirstName, &user.LastName,
		&user.Email, &user.Password, &user.CompanyID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.User{}, err
		}
		slog.Error("error getting user by email", "err", err, "email", email)
		return entity.User{}, err
	}
	return user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, id int, user *entity.User) error {
	// Build dynamic query based on provided fields
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if user.FirstName != "" {
		updates = append(updates, fmt.Sprintf("first_name = $%d", argPos))
		args = append(args, user.FirstName)
		argPos++
	}
	if user.LastName != "" {
		updates = append(updates, fmt.Sprintf("last_name = $%d", argPos))
		args = append(args, user.LastName)
		argPos++
	}
	if user.Email != "" {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, user.Email)
		argPos++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d RETURNING updated_at",
		strings.Join(updates, ", "), argPos)

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&user.UpdatedAt)
	if err != nil {
		slog.Error("error updating user", "err", err, "user_id", id)
		return err
	}
	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		slog.Error("error deleting user", "err", err, "user_id", id)
		return err
	}
	return nil
}

func (r *userRepository) GetUsersByCompanyID(ctx context.Context, companyID int) ([]entity.User, error) {
	query := `SELECT id, role, first_name, last_name, email, password, company_id, created_at, updated_at 
	          FROM users WHERE company_id = $1`
	var users []entity.User
	err := r.db.SelectContext(ctx, &users, query, companyID)
	if err != nil {
		slog.Error("error getting users by company id", "err", err, "company_id", companyID)
		return nil, err
	}
	return users, nil
}
