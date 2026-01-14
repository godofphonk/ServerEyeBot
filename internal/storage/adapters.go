package storage

import (
	"context"

	"github.com/servereye/servereyebot/pkg/domain"
)

// UserRepositoryAdapter adapts PostgreSQL to domain.UserRepository
type UserRepositoryAdapter struct {
	postgres *PostgreSQL
}

// NewUserRepositoryAdapter creates a new user repository adapter
func NewUserRepositoryAdapter(postgres *PostgreSQL) *UserRepositoryAdapter {
	return &UserRepositoryAdapter{postgres: postgres}
}

// Create implements domain.UserRepository
func (a *UserRepositoryAdapter) Create(ctx context.Context, user *domain.User) error {
	return a.postgres.CreateUser(ctx, user)
}

// GetByTelegramID implements domain.UserRepository
func (a *UserRepositoryAdapter) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	return a.postgres.GetUserByTelegramID(ctx, telegramID)
}

// Update implements domain.UserRepository
func (a *UserRepositoryAdapter) Update(ctx context.Context, user *domain.User) error {
	return a.postgres.UpdateUser(ctx, user)
}

// Delete implements domain.UserRepository
func (a *UserRepositoryAdapter) Delete(ctx context.Context, id int) error {
	return a.postgres.DeleteUser(ctx, id)
}

// ServerRepositoryAdapter adapts PostgreSQL to domain.ServerRepository
type ServerRepositoryAdapter struct {
	postgres *PostgreSQL
}

// NewServerRepositoryAdapter creates a new server repository adapter
func NewServerRepositoryAdapter(postgres *PostgreSQL) *ServerRepositoryAdapter {
	return &ServerRepositoryAdapter{postgres: postgres}
}

// Create implements domain.ServerRepository
func (a *ServerRepositoryAdapter) Create(ctx context.Context, server *domain.Server) error {
	return a.postgres.CreateServer(ctx, server)
}

// GetByID implements domain.ServerRepository
func (a *ServerRepositoryAdapter) GetByID(ctx context.Context, id int) (*domain.Server, error) {
	return a.postgres.GetServerByID(ctx, id)
}

// GetByServerID implements domain.ServerRepository
func (a *ServerRepositoryAdapter) GetByServerID(ctx context.Context, serverID string) (*domain.Server, error) {
	return a.postgres.GetServerByServerID(ctx, serverID)
}

// Update implements domain.ServerRepository
func (a *ServerRepositoryAdapter) Update(ctx context.Context, server *domain.Server) error {
	return a.postgres.UpdateServer(ctx, server)
}

// Delete implements domain.ServerRepository
func (a *ServerRepositoryAdapter) Delete(ctx context.Context, id int) error {
	return a.postgres.DeleteServer(ctx, id)
}

// ListByUserID implements domain.ServerRepository
func (a *ServerRepositoryAdapter) ListByUserID(ctx context.Context, userID int) ([]*domain.Server, error) {
	return a.postgres.ListServersByUserID(ctx, userID)
}

// UserServerRepositoryAdapter adapts PostgreSQL to domain.UserServerRepository
type UserServerRepositoryAdapter struct {
	postgres *PostgreSQL
}

// NewUserServerRepositoryAdapter creates a new user-server repository adapter
func NewUserServerRepositoryAdapter(postgres *PostgreSQL) *UserServerRepositoryAdapter {
	return &UserServerRepositoryAdapter{postgres: postgres}
}

// Create implements domain.UserServerRepository
func (a *UserServerRepositoryAdapter) Create(ctx context.Context, userServer *domain.UserServer) error {
	return a.postgres.CreateUserServer(ctx, userServer)
}

// Delete implements domain.UserServerRepository
func (a *UserServerRepositoryAdapter) Delete(ctx context.Context, userID, serverID int) error {
	return a.postgres.DeleteUserServer(ctx, userID, serverID)
}

// GetUserRole implements domain.UserServerRepository
func (a *UserServerRepositoryAdapter) GetUserRole(ctx context.Context, userID, serverID int) (string, error) {
	return a.postgres.GetUserRole(ctx, userID, serverID)
}

// ListServersByUserID implements domain.UserServerRepository
func (a *UserServerRepositoryAdapter) ListServersByUserID(ctx context.Context, userID int) ([]*domain.Server, error) {
	return a.postgres.ListServersByUserID2(ctx, userID)
}

// ListUsersByServerID implements domain.UserServerRepository
func (a *UserServerRepositoryAdapter) ListUsersByServerID(ctx context.Context, serverID int) ([]*domain.User, error) {
	return a.postgres.ListUsersByServerID(ctx, serverID)
}
