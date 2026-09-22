package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redora/redora/backend/internal/models"
)

type ConnectionRepository interface {
	GetAll(ctx context.Context) ([]*models.Connection, error)
	GetByID(ctx context.Context, id string) (*models.Connection, error)
	Create(ctx context.Context, conn *models.Connection) error
	Update(ctx context.Context, conn *models.Connection) error
	Delete(ctx context.Context, id string) error
}

type connectionRepository struct {
	db *sql.DB
}

func NewConnectionRepository(db *sql.DB) ConnectionRepository {
	return &connectionRepository{db: db}
}

func (r *connectionRepository) GetAll(ctx context.Context) ([]*models.Connection, error) {
	query := `SELECT id, name, host, port, username, password_encrypted, db, tls_enabled, created_at, updated_at FROM connections ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []*models.Connection
	for rows.Next() {
		var c models.Connection
		var tlsInt int
		if err := rows.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.PasswordEncrypted, &c.DB, &tlsInt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan connection row: %w", err)
		}
		c.TLSEnabled = tlsInt == 1
		connections = append(connections, &c)
	}
	return connections, nil
}

func (r *connectionRepository) GetByID(ctx context.Context, id string) (*models.Connection, error) {
	query := `SELECT id, name, host, port, username, password_encrypted, db, tls_enabled, created_at, updated_at FROM connections WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var c models.Connection
	var tlsInt int
	if err := row.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.PasswordEncrypted, &c.DB, &tlsInt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan connection by id: %w", err)
	}
	c.TLSEnabled = tlsInt == 1
	return &c, nil
}

func (r *connectionRepository) Create(ctx context.Context, conn *models.Connection) error {
	tlsInt := 0
	if conn.TLSEnabled {
		tlsInt = 1
	}
	query := `INSERT INTO connections (id, name, host, port, username, password_encrypted, db, tls_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, conn.ID, conn.Name, conn.Host, conn.Port, conn.Username, conn.PasswordEncrypted, conn.DB, tlsInt, conn.CreatedAt, conn.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert connection: %w", err)
	}
	return nil
}

func (r *connectionRepository) Update(ctx context.Context, conn *models.Connection) error {
	tlsInt := 0
	if conn.TLSEnabled {
		tlsInt = 1
	}
	query := `UPDATE connections SET name = ?, host = ?, port = ?, username = ?, password_encrypted = ?, db = ?, tls_enabled = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, conn.Name, conn.Host, conn.Port, conn.Username, conn.PasswordEncrypted, conn.DB, tlsInt, conn.UpdatedAt, conn.ID)
	if err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}
	return nil
}

func (r *connectionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM connections WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}
	return nil
}
