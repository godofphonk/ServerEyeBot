package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/servereye/servereyebot/pkg/errors"
)

// Config represents application configuration
type Config struct {
	App        AppConfig        `yaml:"app"`
	Telegram   TelegramConfig   `yaml:"telegram"`
	Logger     LoggerConfig     `yaml:"logger"`
	Metrics    MetricsConfig    `yaml:"metrics"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	API        APIConfig        `yaml:"api"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

// AppConfig represents application configuration
type AppConfig struct {
	Name        string        `yaml:"name"`
	Version     string        `yaml:"version"`
	Environment string        `yaml:"environment"`
	Port        int           `yaml:"port"`
	Timeout     time.Duration `yaml:"timeout"`
	Debug       bool          `yaml:"debug"`
}

// TelegramConfig represents Telegram bot configuration
type TelegramConfig struct {
	Token           string        `yaml:"token"`
	WebhookURL      string        `yaml:"webhook_url"`
	WebhookPort     int           `yaml:"webhook_port"`
	MaxConnections  int           `yaml:"max_connections"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	RateLimitPerSec int           `yaml:"rate_limit_per_sec"`
	RateLimitBurst  int           `yaml:"rate_limit_burst"`
	AdminUserID     int64         `yaml:"admin_user_id"`
	AllowedUserIDs  []int64       `yaml:"allowed_user_ids"`
	PrivateMode     bool          `yaml:"private_mode"`
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"` // json, text
	Output     string `yaml:"output"` // stdout, stderr, file
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"` // days
	Compress   bool   `yaml:"compress"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Interval      time.Duration `yaml:"interval"`
	Retention     time.Duration `yaml:"retention"`
	ExportEnabled bool          `yaml:"export_enabled"`
	ExportFormat  string        `yaml:"export_format"` // prometheus, json
	ExportPort    int           `yaml:"export_port"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Database        string        `yaml:"database"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	URL             string        `yaml:"url"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Password     string        `yaml:"password"`
	Database     int           `yaml:"database"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	Enabled          bool               `yaml:"enabled"`
	CheckInterval    time.Duration      `yaml:"check_interval"`
	AlertThresholds  map[string]float64 `yaml:"alert_thresholds"`
	NotificationURL  string             `yaml:"notification_url"`
	HealthCheckURL   string             `yaml:"health_check_url"`
	MetricsEndpoints []string           `yaml:"metrics_endpoints"`
}

// APIConfig represents ServerEye API configuration
type APIConfig struct {
	BaseURL       string        `yaml:"base_url"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	Enabled       bool          `yaml:"enabled"`
}

// Load loads configuration from environment variables and defaults
func Load() (*Config, error) {
	cfg := &Config{}

	// App configuration
	cfg.App = AppConfig{
		Name:        getEnv("APP_NAME", "ServerEyeBot"),
		Version:     getEnv("APP_VERSION", "1.0.0"),
		Environment: getEnv("ENV", "development"),
		Port:        getEnvInt("PORT", 8080),
		Timeout:     getEnvDuration("APP_TIMEOUT", 30*time.Second),
		Debug:       getEnvBool("DEBUG", false),
	}

	// Telegram configuration
	token := getEnv("TELEGRAM_TOKEN", "")
	if token == "" {
		return nil, errors.NewRequiredFieldError("TELEGRAM_TOKEN")
	}

	cfg.Telegram = TelegramConfig{
		Token:           token,
		WebhookURL:      getEnv("TELEGRAM_WEBHOOK_URL", ""),
		WebhookPort:     getEnvInt("TELEGRAM_WEBHOOK_PORT", 8443),
		MaxConnections:  getEnvInt("TELEGRAM_MAX_CONNECTIONS", 40),
		RequestTimeout:  getEnvDuration("TELEGRAM_REQUEST_TIMEOUT", 10*time.Second),
		RateLimitPerSec: getEnvInt("TELEGRAM_RATE_LIMIT_PER_SEC", 30),
		RateLimitBurst:  getEnvInt("TELEGRAM_RATE_LIMIT_BURST", 10),
		AdminUserID:     getEnvInt64("ADMIN_USER_ID", 0),
		AllowedUserIDs:  getEnvInt64Slice("ALLOWED_USER_IDS", []int64{}),
		PrivateMode:     getEnvBool("TELEGRAM_PRIVATE_MODE", false),
	}

	// Logger configuration
	cfg.Logger = LoggerConfig{
		Level:      getEnv("LOG_LEVEL", "info"),
		Format:     getEnv("LOG_FORMAT", "json"),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		Filename:   getEnv("LOG_FILENAME", "app.log"),
		MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
		MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
		MaxAge:     getEnvInt("LOG_MAX_AGE", 28),
		Compress:   getEnvBool("LOG_COMPRESS", true),
	}

	// Metrics configuration
	cfg.Metrics = MetricsConfig{
		Enabled:       getEnvBool("METRICS_ENABLED", true),
		Interval:      getEnvDuration("METRICS_INTERVAL", 30*time.Second),
		Retention:     getEnvDuration("METRICS_RETENTION", 24*time.Hour),
		ExportEnabled: getEnvBool("METRICS_EXPORT_ENABLED", false),
		ExportFormat:  getEnv("METRICS_EXPORT_FORMAT", "prometheus"),
		ExportPort:    getEnvInt("METRICS_EXPORT_PORT", 9090),
	}

	// Database configuration
	cfg.Database = DatabaseConfig{
		Driver:          getEnv("DB_DRIVER", "postgres"),
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		Database:        getEnv("DB_NAME", "servereye"),
		Username:        getEnv("DB_USERNAME", "servereye"),
		Password:        getEnv("DB_PASSWORD", "servereye123"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		URL:             getEnv("DATABASE_URL", "postgres://servereye:servereye123@localhost:5432/servereye?sslmode=disable"),
	}

	// Redis configuration (optional for now)
	cfg.Redis = RedisConfig{
		Host:         getEnv("REDIS_HOST", "localhost"),
		Port:         getEnvInt("REDIS_PORT", 6379),
		Password:     getEnv("REDIS_PASSWORD", ""),
		Database:     getEnvInt("REDIS_DATABASE", 0),
		PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		MaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
		DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
	}

	// Monitoring configuration
	cfg.Monitoring = MonitoringConfig{
		Enabled:          getEnvBool("MONITORING_ENABLED", true),
		CheckInterval:    getEnvDuration("MONITORING_CHECK_INTERVAL", 30*time.Second),
		AlertThresholds:  getEnvFloatMap("MONITORING_ALERT_THRESHOLDS", map[string]float64{}),
		NotificationURL:  getEnv("MONITORING_NOTIFICATION_URL", ""),
		HealthCheckURL:   getEnv("MONITORING_HEALTH_CHECK_URL", ""),
		MetricsEndpoints: getEnvStringSlice("MONITORING_METRICS_ENDPOINTS", []string{}),
	}

	// API configuration
	cfg.API = APIConfig{
		BaseURL:       getEnv("API_BASE_URL", "http://localhost:8080"),
		Timeout:       getEnvDuration("API_TIMEOUT", 30*time.Second),
		RetryAttempts: getEnvInt("API_RETRY_ATTEMPTS", 3),
		RetryDelay:    getEnvDuration("API_RETRY_DELAY", 1*time.Second),
		Enabled:       getEnvBool("API_ENABLED", true),
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Telegram.Token == "" {
		return errors.NewRequiredFieldError("TELEGRAM_TOKEN")
	}

	if c.App.Port <= 0 || c.App.Port > 65535 {
		return errors.NewValidationError("invalid port number", map[string]interface{}{"port": c.App.Port})
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.Logger.Level] {
		return errors.NewValidationError("invalid log level", map[string]interface{}{"level": c.Logger.Level})
	}

	return nil
}

// String returns a string representation of the configuration (without sensitive data)
func (c *Config) String() string {
	return fmt.Sprintf("Config{App: %+v, Telegram: {Token: [REDACTED], ...}, Logger: %+v}", c.App, c.Logger)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getEnvInt64Slice(key string, defaultValue []int64) []int64 {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]int64, 0, len(parts))
		for _, part := range parts {
			if intVal, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
				result = append(result, intVal)
			}
		}
		return result
	}
	return defaultValue
}

func getEnvFloatMap(key string, defaultValue map[string]float64) map[string]float64 {
	if value := os.Getenv(key); value != "" {
		result := make(map[string]float64)
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				if floatVal, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64); err == nil {
					result[strings.TrimSpace(kv[0])] = floatVal
				}
			}
		}
		return result
	}
	return defaultValue
}
