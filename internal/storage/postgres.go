package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/servereye/servereyebot/pkg/domain"
)

// PostgreSQL implements database operations using PostgreSQL
type PostgreSQL struct {
	db *sql.DB
}

// NewPostgreSQL creates a new PostgreSQL instance
func NewPostgreSQL(databaseURL string) (*PostgreSQL, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgreSQL{db: db}, nil
}

// Close closes the database connection
func (p *PostgreSQL) Close() error {
	return p.db.Close()
}

// UserRepository implementation

// CreateUser creates a new user in the database
func (p *PostgreSQL) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	now := time.Now()
	err := p.db.QueryRowContext(ctx, query,
		user.TelegramID, user.Username, user.FirstName, user.LastName, true, now, now).
		Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByTelegramID retrieves a user by their Telegram ID
func (p *PostgreSQL) GetUserByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_active, created_at, updated_at
		FROM users
		WHERE telegram_id = $1 AND is_active = true`

	var user domain.User
	var createdAt, updatedAt time.Time

	err := p.db.QueryRowContext(ctx, query, telegramID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
		&user.IsAdmin, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt = createdAt
	user.LastSeen = updatedAt

	return &user, nil
}

// UpdateUser updates an existing user
func (p *PostgreSQL) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET username = $2, first_name = $3, last_name = $4, updated_at = $5
		WHERE id = $1`

	_, err := p.db.ExecContext(ctx, query,
		user.ID, user.Username, user.FirstName, user.LastName, time.Now())

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user
func (p *PostgreSQL) DeleteUser(ctx context.Context, id int) error {
	query := `UPDATE users SET is_active = false WHERE id = $1`

	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ServerRepository implementation

// CreateServer creates a new server in the database
func (p *PostgreSQL) CreateServer(ctx context.Context, server *domain.Server) error {
	query := `
		INSERT INTO servers (server_id, name, description, ip_address, port, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	now := time.Now()
	err := p.db.QueryRowContext(ctx, query,
		server.ServerID, server.Name, server.Description, server.IPAddress,
		server.Port, server.IsActive, now, now).Scan(&server.ID)

	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	server.CreatedAt = now
	server.UpdatedAt = now

	return nil
}

// GetServerByID retrieves a server by its database ID
func (p *PostgreSQL) GetServerByID(ctx context.Context, id int) (*domain.Server, error) {
	query := `
		SELECT id, server_id, name, description, ip_address, port, is_active, created_at, updated_at
		FROM servers
		WHERE id = $1 AND is_active = true`

	var server domain.Server
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID, &server.ServerID, &server.Name, &server.Description,
		&server.IPAddress, &server.Port, &server.IsActive,
		&server.CreatedAt, &server.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// GetServerByServerID retrieves a server by its server_id (e.g., srv_12313)
func (p *PostgreSQL) GetServerByServerID(ctx context.Context, serverID string) (*domain.Server, error) {
	query := `
		SELECT id, server_id, name, description, ip_address, port, is_active, created_at, updated_at
		FROM servers
		WHERE server_id = $1 AND is_active = true`

	var server domain.Server
	err := p.db.QueryRowContext(ctx, query, serverID).Scan(
		&server.ID, &server.ServerID, &server.Name, &server.Description,
		&server.IPAddress, &server.Port, &server.IsActive,
		&server.CreatedAt, &server.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// UpdateServer updates an existing server
func (p *PostgreSQL) UpdateServer(ctx context.Context, server *domain.Server) error {
	query := `
		UPDATE servers
		SET name = $2, description = $3, ip_address = $4, port = $5, is_active = $6, updated_at = $7
		WHERE id = $1`

	server.UpdatedAt = time.Now()
	_, err := p.db.ExecContext(ctx, query,
		server.ID, server.Name, server.Description, server.IPAddress,
		server.Port, server.IsActive, server.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	return nil
}

// DeleteServer soft deletes a server
func (p *PostgreSQL) DeleteServer(ctx context.Context, id int) error {
	query := `UPDATE servers SET is_active = false WHERE id = $1`

	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}

// ListServersByUserID retrieves all servers associated with a user
func (p *PostgreSQL) ListServersByUserID(ctx context.Context, userID int) ([]*domain.Server, error) {
	query := `
		SELECT s.id, s.server_id, s.name, s.description, s.ip_address, s.port, s.is_active, s.created_at, s.updated_at
		FROM servers s
		INNER JOIN user_servers us ON s.id = us.server_id
		INNER JOIN users u ON us.user_id = u.id
		WHERE u.telegram_id = $1 AND s.is_active = true AND u.is_active = true
		ORDER BY s.name`

	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	var servers []*domain.Server
	for rows.Next() {
		var server domain.Server
		err := rows.Scan(
			&server.ID, &server.ServerID, &server.Name, &server.Description,
			&server.IPAddress, &server.Port, &server.IsActive,
			&server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, &server)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating servers: %w", err)
	}

	return servers, nil
}

// UserServerRepository implementation

// CreateUserServer creates a new user-server relationship
func (p *PostgreSQL) CreateUserServer(ctx context.Context, userServer *domain.UserServer) error {
	query := `
		INSERT INTO user_servers (user_id, server_id, role, added_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			role = EXCLUDED.role,
			added_at = EXCLUDED.added_at
		RETURNING id`

	userServer.AddedAt = time.Now()
	err := p.db.QueryRowContext(ctx, query,
		userServer.UserID, userServer.ServerID, userServer.Role, userServer.AddedAt).
		Scan(&userServer.ID)

	if err != nil {
		return fmt.Errorf("failed to create user-server relationship: %w", err)
	}

	return nil
}

// DeleteUserServer removes a user-server relationship
func (p *PostgreSQL) DeleteUserServer(ctx context.Context, userID, serverID int) error {
	query := `DELETE FROM user_servers WHERE user_id = $1 AND server_id = $2`

	_, err := p.db.ExecContext(ctx, query, userID, serverID)
	if err != nil {
		return fmt.Errorf("failed to delete user-server relationship: %w", err)
	}

	return nil
}

// GetUserRole retrieves the role of a user for a specific server
func (p *PostgreSQL) GetUserRole(ctx context.Context, userID, serverID int) (string, error) {
	query := `
		SELECT role
		FROM user_servers
		WHERE user_id = $1 AND server_id = $2`

	var role string
	err := p.db.QueryRowContext(ctx, query, userID, serverID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user-server relationship not found")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// ListServersByUserID retrieves all servers for a user (alias for ListServersByUserID)
func (p *PostgreSQL) ListServersByUserID2(ctx context.Context, userID int) ([]*domain.Server, error) {
	return p.ListServersByUserID(ctx, userID)
}

// ListUsersByServerID retrieves all users associated with a server
func (p *PostgreSQL) ListUsersByServerID(ctx context.Context, serverID int) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.telegram_id, u.username, u.first_name, u.last_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_servers us ON u.id = us.user_id
		WHERE us.server_id = $1 AND u.is_active = true
		ORDER BY u.username`

	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.IsAdmin, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.CreatedAt = createdAt
		user.LastSeen = updatedAt
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}
