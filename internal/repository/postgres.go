package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/servereye/servereyebot/internal/models"
)

// PostgresRepository implements database operations
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(databaseURL string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresRepository{db: db}, nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// CreateUser creates a new user
func (r *PostgresRepository) CreateUser(user *models.User) error {
	query := `
INSERT INTO users (telegram_id, username, first_name, last_name, is_admin, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (telegram_id) DO UPDATE SET
username = EXCLUDED.username,
first_name = EXCLUDED.first_name,
last_name = EXCLUDED.last_name,
is_admin = EXCLUDED.is_admin,
updated_at = CURRENT_TIMESTAMP
RETURNING id
`

	var returnedID int64
	err := r.db.QueryRow(query, user.TelegramID, user.Username, user.FirstName, user.LastName, user.IsAdmin, user.IsActive).Scan(&returnedID)
	if err == nil {
		user.ID = returnedID
	}
	return err
}

// GetUser retrieves a user by ID
func (r *PostgresRepository) GetUser(userID int64) (*models.User, error) {
	query := `
SELECT id, telegram_id, username, first_name, last_name, is_admin, is_active, created_at, updated_at
FROM users WHERE telegram_id = $1
`

	var user models.User
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
		&user.IsAdmin, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// AddServerToUser adds a server to a user's server list
func (r *PostgresRepository) AddServerToUser(userID int64, serverID, source string) error {
	// First, ensure the server exists
	if err := r.ensureServerExists(serverID); err != nil {
		return err
	}

	// Then add the relationship
	query := `
INSERT INTO user_servers (user_id, server_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, server_id) DO NOTHING
`

	_, err := r.db.Exec(query, userID, serverID, "viewer")
	return err
}

// ensureServerExists creates a server if it doesn't exist
func (r *PostgresRepository) ensureServerExists(serverID string) error {
	query := `
INSERT INTO servers (server_id, name, description)
VALUES ($1, $1, '')
ON CONFLICT (server_id) DO NOTHING
`

	_, err := r.db.Exec(query, serverID)
	return err
}

// GetUserServers retrieves all servers for a user
func (r *PostgresRepository) GetUserServers(userID int64) ([]models.ServerWithDetails, error) {
	query := `
SELECT s.server_id as id, s.name, s.description, s.created_at, s.updated_at,
       s.server_id as server_key, us.role as source, us.added_at
FROM servers s
INNER JOIN user_servers us ON s.server_id = us.server_id
WHERE us.user_id = $1
ORDER BY us.added_at DESC
`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []models.ServerWithDetails
	for rows.Next() {
		var server models.ServerWithDetails
		err := rows.Scan(
			&server.ID, &server.Name, &server.Description,
			&server.CreatedAt, &server.UpdatedAt,
			&server.ServerKey, &server.Role, &server.AddedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// RemoveServerFromUser removes a server from a user's server list
func (r *PostgresRepository) RemoveServerFromUser(userID int64, serverID string) error {
	query := `DELETE FROM user_servers WHERE user_id = $1 AND server_id = $2`
	_, err := r.db.Exec(query, userID, serverID)
	return err
}

// IsServerOwnedByUser checks if a server is owned by a user
func (r *PostgresRepository) IsServerOwnedByUser(userID int64, serverID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_servers WHERE user_id = $1 AND server_id = $2)`

	var exists bool
	err := r.db.QueryRow(query, userID, serverID).Scan(&exists)
	return exists, err
}
