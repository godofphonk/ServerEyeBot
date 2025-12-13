package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBotConfig_Valid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bot.yaml")

	validConfig := `
telegram:
  token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

database:
  url: "postgres://user:pass@localhost/dbname"

logging:
  level: "info"
  file: "/var/log/bot.log"
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadBotConfig(configPath)
	if err != nil {
		t.Fatalf("LoadBotConfig() error = %v", err)
	}

	if config.Telegram.Token != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Errorf("Telegram.Token = %v", config.Telegram.Token)
	}
	if config.Database.URL != "postgres://user:pass@localhost/dbname" {
		t.Errorf("Database.URL = %v", config.Database.URL)
	}
}

func TestBotConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  BotConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BotConfig{
				Telegram: TelegramConfig{Token: "test_token"},
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
			},
			wantErr: false,
		},
		{
			name: "missing telegram token",
			config: BotConfig{
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
			},
			wantErr: true,
		},
		{
			name: "missing database url",
			config: BotConfig{
				Telegram: TelegramConfig{Token: "test_token"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadBotConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `
telegram:
  token: "broken
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err := LoadBotConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
