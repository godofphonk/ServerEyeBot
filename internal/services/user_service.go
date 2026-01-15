package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/servereye/servereyebot/internal/api"
	"github.com/servereye/servereyebot/internal/models"
	"github.com/servereye/servereyebot/internal/repository"
)

// UserService handles user and server operations
type UserService struct {
	repo      *repository.PostgresRepository
	apiClient *api.Client
}

// NewUserService creates a new user service
func NewUserService(repo *repository.PostgresRepository, apiClient *api.Client) *UserService {
	return &UserService{repo: repo, apiClient: apiClient}
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

// AddServerToUser adds a server to user's server list with proper API validation
func (s *UserService) AddServerToUser(ctx context.Context, userID int64, serverKey, source string) error {
	log.Printf("Adding server %s to user %d", serverKey, userID)

	// Validate server key format
	if err := api.ValidateServerID(serverKey); err != nil {
		log.Printf("Invalid server key format: %v", err)
		return err
	}

	// Check if server exists and get its sources
	if s.apiClient != nil {
		sourcesResp, err := s.apiClient.GetServerSources(ctx, serverKey)
		if err != nil {
			log.Printf("Server validation failed for %s: %v", serverKey, err)
			// Check if it's a "not found" error
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
				return fmt.Errorf("server '%s' not found", serverKey)
			}
			// For other API errors, return a more specific message
			return fmt.Errorf("failed to validate server '%s': API error", serverKey)
		}

		log.Printf("Server %s found with ID %s, sources: %v", serverKey, sourcesResp.ServerID, sourcesResp.Sources)

		// Check if TGBot is already in sources
		hasTGBot := false
		for _, src := range sourcesResp.Sources {
			if src == "TGBot" {
				hasTGBot = true
				break
			}
		}

		// If TGBot is not in sources, add it
		if !hasTGBot {
			log.Printf("TGBot source not found for server %s, adding it...", serverKey)
			_, err := s.apiClient.AddServerSourceByRequest(ctx, serverKey)
			if err != nil {
				log.Printf("Failed to add TGBot source to server %s: %v", serverKey, err)
				return fmt.Errorf("failed to add TGBot source to server '%s'", serverKey)
			}
			log.Printf("TGBot source added successfully to server %s", serverKey)
		} else {
			log.Printf("TGBot source already exists for server %s", serverKey)
		}

		// Use the original serverKey for database storage (not ServerID from API)
		return s.repo.AddServerToUser(userID, serverKey, source)
	} else {
		log.Printf("API client not available, skipping server validation")
		return s.repo.AddServerToUser(userID, serverKey, source)
	}
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

	result := fmt.Sprintf("*Ваши серверы (%d):*\n\n", len(servers))

	for i, server := range servers {
		result += fmt.Sprintf("%d. `%s`", i+1, server.ID)

		if server.Name != server.ID {
			result += fmt.Sprintf(" - %s", server.Name)
		}

		result += fmt.Sprintf("\nДобавлен: %s\n", server.AddedAt.Format("02.01.2006 15:04"))
		result += fmt.Sprintf("Источник: %s\n\n", server.Source)
	}

	return result
}

// FormatServersListPlain formats servers list for display without Markdown
func (s *UserService) FormatServersListPlain(servers []models.ServerWithDetails) string {
	if len(servers) == 0 {
		return "У вас пока нет добавленных серверов.\n\nИспользуйте команду /add <server_id> чтобы добавить сервер."
	}

	result := fmt.Sprintf("Ваши серверы (%d):\n\n", len(servers))

	for i, server := range servers {
		result += fmt.Sprintf("%d. %s", i+1, server.ID)

		if server.Name != server.ID {
			result += fmt.Sprintf(" - %s", server.Name)
		}

		result += fmt.Sprintf("\nДобавлен: %s\n", server.AddedAt.Format("02.01.2006 15:04"))
		result += fmt.Sprintf("Источник: %s\n\n", server.Source)
	}

	return result
}
