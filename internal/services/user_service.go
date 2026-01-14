package services

import (
"context"
"fmt"
"log"

"github.com/servereye/servereyebot/internal/models"
"github.com/servereye/servereyebot/internal/repository"
)

// UserService handles user and server operations
type UserService struct {
repo *repository.PostgresRepository
}

// NewUserService creates a new user service
func NewUserService(repo *repository.PostgresRepository) *UserService {
return &UserService{repo: repo}
}

// RegisterOrUpdateUser registers a new user or updates existing one
func (s *UserService) RegisterOrUpdateUser(ctx context.Context, user *models.User) error {
log.Printf("Registering user: %d (%s)", user.ID, user.Username)
return s.repo.CreateUser(user)
}

// GetUser retrieves user by ID
func (s *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
return s.repo.GetUser(userID)
}

// AddServerToUser adds a server to user's server list
func (s *UserService) AddServerToUser(ctx context.Context, userID int64, serverID, source string) error {
log.Printf("Adding server %s to user %d", serverID, userID)
return s.repo.AddServerToUser(userID, serverID, source)
}

// GetUserServers retrieves all servers for a user
func (s *UserService) GetUserServers(ctx context.Context, userID int64) ([]models.ServerWithDetails, error) {
log.Printf("Getting servers for user %d", userID)
return s.repo.GetUserServers(userID)
}

// RemoveServerFromUser removes a server from user's server list
func (s *UserService) RemoveServerFromUser(ctx context.Context, userID int64, serverID string) error {
log.Printf("Removing server %s from user %d", serverID, userID)
return s.repo.RemoveServerFromUser(userID, serverID)
}

// IsServerOwnedByUser checks if server is owned by user
func (s *UserService) IsServerOwnedByUser(ctx context.Context, userID int64, serverID string) (bool, error) {
return s.repo.IsServerOwnedByUser(userID, serverID)
}

// FormatServersList formats servers list for display
func (s *UserService) FormatServersList(servers []models.ServerWithDetails) string {
if len(servers) == 0 {
return "У вас пока нет добавленных серверов.\n\nИспользуйте команду /add <server_id> чтобы добавить сервер."
}

result := fmt.Sprintf("�� **Ваши серверы (%d):**\n\n", len(servers))

for i, server := range servers {
result += fmt.Sprintf("%d. `%s`", i+1, server.ID)

if server.Name != server.ID {
result += fmt.Sprintf(" - %s", server.Name)
}

result += fmt.Sprintf("\n   📅 Добавлен: %s\n", server.AddedAt.Format("02.01.2006 15:04"))
result += fmt.Sprintf("   🔗 Источник: %s\n\n", server.Source)
}

return result
}
