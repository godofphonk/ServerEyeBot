package services

import (
	"context"

	"github.com/servereye/servereyebot/internal/models"
	"github.com/servereye/servereyebot/pkg/domain"
)

// UserServiceAdapter adapts our UserService to domain.UserService
type UserServiceAdapter struct {
	service *UserService
}

// NewUserServiceAdapter creates a new adapter
func NewUserServiceAdapter(service *UserService) *UserServiceAdapter {
	return &UserServiceAdapter{service: service}
}

// IsAdmin checks if user is admin
func (a *UserServiceAdapter) IsAdmin(userID int64) bool {
	// For now, check against environment variable
	// TODO: Implement proper admin checking
	return userID == 1805441944 // hardcoded for now
}

// IsAuthorized checks if user is authorized
func (a *UserServiceAdapter) IsAuthorized(userID int64) bool {
	// For now, all users are authorized
	// TODO: Implement proper authorization
	return true
}

// RegisterUser registers a user
func (a *UserServiceAdapter) RegisterUser(ctx context.Context, user *domain.User) error {
	modelUser := &models.User{
		ID:         0,               // Let database generate ID
		TelegramID: user.TelegramID, // Store TelegramID separately
		Username:   user.Username,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		IsAdmin:    a.IsAdmin(user.TelegramID),
		IsActive:   true,
	}
	return a.service.RegisterOrUpdateUser(ctx, modelUser)
}

// GetUser retrieves a user
func (a *UserServiceAdapter) GetUser(ctx context.Context, userID int64) (*domain.User, error) {
	modelUser, err := a.service.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:         int(modelUser.ID),    // Convert int64 to int for domain.User
		TelegramID: modelUser.TelegramID, // Use actual TelegramID from database
		Username:   modelUser.Username,
		FirstName:  modelUser.FirstName,
		LastName:   modelUser.LastName,
		IsAdmin:    modelUser.IsAdmin,
		CreatedAt:  modelUser.CreatedAt,
		LastSeen:   modelUser.UpdatedAt, // Use updated_at as last_seen
	}, nil
}

// GetUserServers retrieves all servers for a user
func (a *UserServiceAdapter) GetUserServers(ctx context.Context, userID int64) ([]models.ServerWithDetails, error) {
	return a.service.GetUserServers(ctx, userID)
}

// AddServerToUser adds a server to user's server list
func (a *UserServiceAdapter) AddServerToUser(ctx context.Context, userID int64, serverID, source string) error {
	return a.service.AddServerToUser(ctx, userID, serverID, source)
}

// FormatServersList formats servers list for display
func (a *UserServiceAdapter) FormatServersList(servers []models.ServerWithDetails) string {
	return a.service.FormatServersList(servers)
}
