package domain

import (
	"context"
	"time"
)

// SystemMetrics represents system monitoring data
type SystemMetrics struct {
	CPU     CPUMetrics     `json:"cpu"`
	Memory  MemoryMetrics  `json:"memory"`
	Disk    DiskMetrics    `json:"disk"`
	Uptime  UptimeMetrics  `json:"uptime"`
	Network NetworkMetrics `json:"network"`
}

// CPUMetrics represents CPU information
type CPUMetrics struct {
	Temperature float64 `json:"temperature"` // Celsius
	Usage       float64 `json:"usage"`       // Percentage
	Cores       int     `json:"cores"`
	Model       string  `json:"model"`
}

// MemoryMetrics represents memory information
type MemoryMetrics struct {
	Total     uint64  `json:"total"`     // Bytes
	Used      uint64  `json:"used"`      // Bytes
	Available uint64  `json:"available"` // Bytes
	Usage     float64 `json:"usage"`     // Percentage
}

// DiskMetrics represents disk information
type DiskMetrics struct {
	Filesystems []Filesystem `json:"filesystems"`
}

// Filesystem represents a single filesystem
type Filesystem struct {
	Path    string  `json:"path"`
	Total   uint64  `json:"total"` // Bytes
	Used    uint64  `json:"used"`  // Bytes
	Free    uint64  `json:"free"`  // Bytes
	Usage   float64 `json:"usage"` // Percentage
	Fstype  string  `json:"fstype"`
	Mounted bool    `json:"mounted"`
}

// UptimeMetrics represents system uptime
type UptimeMetrics struct {
	Seconds   uint64 `json:"seconds"`
	Days      int    `json:"days"`
	Hours     int    `json:"hours"`
	Minutes   int    `json:"minutes"`
	Formatted string `json:"formatted"`
}

// NetworkMetrics represents network information
type NetworkMetrics struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
	Up        bool   `json:"up"`
}

// MetricsService defines the interface for system metrics collection
type MetricsService interface {
	GetCPU(ctx context.Context) (*CPUMetrics, error)
	GetMemory(ctx context.Context) (*MemoryMetrics, error)
	GetDisk(ctx context.Context) (*DiskMetrics, error)
	GetUptime(ctx context.Context) (*UptimeMetrics, error)
	GetNetwork(ctx context.Context) (*NetworkMetrics, error)
	GetAll(ctx context.Context) (*SystemMetrics, error)
}

// TelegramService defines the interface for Telegram operations
type TelegramService interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard interface{}) error
	StartReceivingUpdates(ctx context.Context, handler interface{}) error
	StopReceivingUpdates()
	AnswerCallback(ctx context.Context, callbackID, text string) error
	AnswerCallbackQuery(ctx context.Context, callbackID, text string) error
	EditMessage(ctx context.Context, chatID int64, messageID int, text string, keyboard interface{}) error
	SetCommands(ctx context.Context, commands []BotCommand) error
}

// BotCommand represents a Telegram bot command
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// UserService defines the interface for user management
type UserService interface {
	IsAdmin(userID int64) bool
	IsAuthorized(userID int64) bool
	RegisterUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, userID int64) (*User, error)
}

// User represents a Telegram user
type User struct {
	ID         int       `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	IsAdmin    bool      `json:"is_admin"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeen   time.Time `json:"last_seen"`
}

// Command represents a bot command
type Command struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Handler     CommandHandler      `json:"-"`
	Middleware  []CommandMiddleware `json:"-"`
	Permissions []string            `json:"permissions"`
}

// CommandHandler defines the function signature for command handlers
type CommandHandler func(ctx context.Context, cmd *Command, args []string) error

// CommandMiddleware defines the function signature for command middleware
type CommandMiddleware func(ctx context.Context, cmd *Command, args []string, next CommandHandler) error

// Event represents a system event
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    int64       `json:"user_id,omitempty"`
	ChatID    int64       `json:"chat_id,omitempty"`
}

// EventBus defines the interface for event handling
type EventBus interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler defines the function signature for event handlers
type EventHandler func(ctx context.Context, event *Event) error

// Server represents a monitored server
type Server struct {
	ID          int       `json:"id"`
	ServerID    string    `json:"server_id"` // e.g., srv_12313
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IPAddress   string    `json:"ip_address"`
	Port        int       `json:"port"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserServer represents the relationship between users and servers
type UserServer struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ServerID  string    `json:"server_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// CallbackQuery represents a callback query from inline keyboard
type CallbackQuery struct {
	ID      string  `json:"id"`
	From    User    `json:"from"`
	Message Message `json:"message"`
	Data    string  `json:"data"`
}

// Message represents a telegram message
type Message struct {
	MessageID int  `json:"message_id"`
	Chat      Chat `json:"chat"`
}

// Chat represents a telegram chat
type Chat struct {
	ID int64 `json:"id"`
}

// ServerWithDetails represents server with user relationship info
type ServerWithDetails struct {
	Server
	Role    string    `json:"role"`
	AddedAt time.Time `json:"added_at"`
}

// UserRepository defines the interface for user database operations
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int) error
}

// ServerRepository defines the interface for server database operations
type ServerRepository interface {
	Create(ctx context.Context, server *Server) error
	GetByID(ctx context.Context, id int) (*Server, error)
	GetByServerID(ctx context.Context, serverID string) (*Server, error)
	Update(ctx context.Context, server *Server) error
	Delete(ctx context.Context, id int) error
	ListByUserID(ctx context.Context, userID int) ([]*Server, error)
}

// UserServerRepository defines the interface for user-server relationship operations
type UserServerRepository interface {
	Create(ctx context.Context, userServer *UserServer) error
	Delete(ctx context.Context, userID, serverID int) error
	GetUserRole(ctx context.Context, userID, serverID int) (string, error)
	ListServersByUserID(ctx context.Context, userID int) ([]*Server, error)
	ListUsersByServerID(ctx context.Context, serverID int) ([]*User, error)
}
