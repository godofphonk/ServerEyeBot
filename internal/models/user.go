package models

import (
	"time"
)

// User represents a user in the database
type User struct {
	ID         int64     `json:"id" db:"id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	Username   string    `json:"username" db:"username"`
	FirstName  string    `json:"first_name" db:"first_name"`
	LastName   string    `json:"last_name" db:"last_name"`
	IsAdmin    bool      `json:"is_admin" db:"is_admin"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Server represents a server in the database
type Server struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserServer represents the relationship between users and servers
type UserServer struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	ServerID  string    `json:"server_id" db:"server_id"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ServerWithDetails represents server with user relationship info
type ServerWithDetails struct {
	Server
	Role      string    `json:"role"`
	AddedAt   time.Time `json:"added_at"`
	ServerKey string    `json:"server_key"` // API key for metrics
}
