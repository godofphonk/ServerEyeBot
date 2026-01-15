package service

import (
	"context"
	"fmt"

	"github.com/servereye/servereyebot/pkg/domain"
)

// ServerService handles server-related operations
type ServerService struct {
	serverRepo     domain.ServerRepository
	userRepo       domain.UserRepository
	userServerRepo domain.UserServerRepository
}

// NewServerService creates a new ServerService instance
func NewServerService(
	serverRepo domain.ServerRepository,
	userRepo domain.UserRepository,
	userServerRepo domain.UserServerRepository,
) *ServerService {
	return &ServerService{
		serverRepo:     serverRepo,
		userRepo:       userRepo,
		userServerRepo: userServerRepo,
	}
}

// ListUserServers returns all servers associated with a user
func (s *ServerService) ListUserServers(ctx context.Context, telegramUserID int64) ([]*domain.Server, error) {
	// First, get the user
	user, err := s.userRepo.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get all servers for this user
	servers, err := s.serverRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	return servers, nil
}

// AddServerToUser adds a server to a user's server list
func (s *ServerService) AddServerToUser(ctx context.Context, telegramUserID int64, serverID string, role string) error {
	// Get the user
	user, err := s.userRepo.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get or create the server
	server, err := s.serverRepo.GetByServerID(ctx, serverID)
	if err != nil {
		// Server doesn't exist, create a new one
		server = &domain.Server{
			ServerID: serverID,
			Name:     serverID, // Default name to server ID
			IsActive: true,
		}
		if err := s.serverRepo.Create(ctx, server); err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}
	}

	// Create the user-server relationship
	userServer := &domain.UserServer{
		UserID:   user.ID,
		ServerID: server.ServerID,
		Role:     role,
	}

	if err := s.userServerRepo.Create(ctx, userServer); err != nil {
		return fmt.Errorf("failed to add server to user: %w", err)
	}

	return nil
}

// RemoveServerFromUser removes a server from a user's server list
func (s *ServerService) RemoveServerFromUser(ctx context.Context, telegramUserID int64, serverID string) error {
	// Get the user
	user, err := s.userRepo.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get the server
	server, err := s.serverRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Remove the user-server relationship
	if err := s.userServerRepo.Delete(ctx, user.ID, server.ID); err != nil {
		return fmt.Errorf("failed to remove server from user: %w", err)
	}

	return nil
}

// FormatServersForMessage formats servers list for Telegram message
func (s *ServerService) FormatServersForMessage(servers []*domain.Server) string {
	if len(servers) == 0 {
		return "У вас нет добавленных серверов. Используйте команду /add чтобы добавить сервер."
	}

	message := "🖥️ *Ваши серверы:*\n\n"
	for i, server := range servers {
		status := "🟢"
		if !server.IsActive {
			status = "🔴"
		}

		message += fmt.Sprintf("%d. %s `%s`\n", i+1, status, server.ServerID)
		if server.Name != server.ServerID {
			message += fmt.Sprintf("   📝 %s\n", server.Name)
		}
		if server.Description != "" {
			message += fmt.Sprintf("   📄 %s\n", server.Description)
		}
		if server.IPAddress != "" {
			message += fmt.Sprintf("   🌐 %s:%d\n", server.IPAddress, server.Port)
		}
		message += "\n"
	}

	return message
}
