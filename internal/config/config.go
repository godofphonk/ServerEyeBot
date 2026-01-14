package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BotConfig конфигурация бота
type BotConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Logging  LoggingConfig  `yaml:"logging"`
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
	return nil
}
