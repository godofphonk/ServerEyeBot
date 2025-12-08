package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BotConfig конфигурация бота
type BotConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Redis    RedisConfig    `yaml:"redis"`
	Database DatabaseConfig `yaml:"database"`
	Kafka    KafkaConfig    `yaml:"kafka,omitempty"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	Token string `yaml:"token"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	URL         string `yaml:"url"`
	KeysURL     string `yaml:"keys_url"`
}

// KafkaConfig конфигурация Kafka
type KafkaConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Brokers      []string `yaml:"brokers"`
	TopicPrefix  string   `yaml:"topic_prefix"`
	Compression  string   `yaml:"compression"`
	MaxAttempts  int      `yaml:"max_attempts"`
	BatchSize    int      `yaml:"batch_size"`
	RequiredAcks int      `yaml:"required_acks"`
}

// LoadBotConfig загружает конфигурацию бота
func LoadBotConfig(filepath string) (*BotConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %v", err)
	}

	var config BotConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("не удалось парсить конфигурацию: %v", err)
	}

	// Load KEYS_DATABASE_URL from environment if not set in YAML
	if config.Database.KeysURL == "" {
		config.Database.KeysURL = os.Getenv("KEYS_DATABASE_URL")
	}

	// Валидация конфигурации
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("некорректная конфигурация: %v", err)
	}

	return &config, nil
}

// validate валидирует конфигурацию бота
func (c *BotConfig) validate() error {
	if c.Telegram.Token == "" {
		return fmt.Errorf("токен Telegram бота не может быть пустым")
	}
	if c.Redis.Address == "" {
		return fmt.Errorf("адрес Redis не может быть пустым")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("URL базы данных не может быть пустым")
	}
	return nil
}
