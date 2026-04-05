package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/servereye/servereyebot/pkg/domain"
	"github.com/servereye/servereyebot/pkg/errors"
)

// Client represents ServerEye API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     Logger
}

// Logger interface for API client
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewClient creates a new API client
func NewClient(baseURL string, logger Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// AddServerSourceRequest represents request to add server source
type AddServerSourceRequest struct {
	Source string `json:"source"` // "TGBot" or "Web"
}

// AddIdentifierRequest represents request to add identifier to server source
type AddIdentifierRequest struct {
	SourceType     string            `json:"source_type"`     // "TGBot"
	Identifiers    []string          `json:"identifiers"`     // Telegram IDs
	IdentifierType string            `json:"identifier_type"` // "telegram_id"
	Metadata       map[string]string `json:"metadata"`        // User metadata
}

// AddIdentifierResponse represents response from adding identifier
type AddIdentifierResponse struct {
	Message        string   `json:"message"`
	ServerID       string   `json:"server_id"`
	ServerKey      string   `json:"server_key"`
	SourceType     string   `json:"source_type"`
	Identifiers    []string `json:"identifiers"`
	IdentifierType string   `json:"identifier_type"`
}

// AddServerSourceResponse represents response from adding server source
type AddServerSourceResponse struct {
	ServerID string `json:"server_id"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

// GetServerSourcesResponse represents response from getting server sources
type GetServerSourcesResponse struct {
	ServerID  string   `json:"server_id"`
	ServerKey string   `json:"server_key"`
	Sources   []string `json:"sources"`
}

// GetServerSources gets server sources by server key
func (c *Client) GetServerSources(ctx context.Context, serverKey string) (*GetServerSourcesResponse, error) {
	c.logger.Debug("Getting server sources", "server_key", serverKey)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources", c.baseURL, serverKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get server sources", "error", err, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", "get server sources", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response GetServerSourcesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server sources retrieved successfully",
		"server_id", response.ServerID,
		"server_key", response.ServerKey,
		"sources_count", len(response.Sources))

	return &response, nil
}

// AddServerSourceByRequest adds TGBot source to server by key
func (c *Client) AddServerSourceByRequest(ctx context.Context, serverKey string) (*AddServerSourceResponse, error) {
	c.logger.Debug("Adding server source by key", "server_key", serverKey, "source", "TGBot")

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources", c.baseURL, serverKey)

	reqBody := AddServerSourceRequest{
		Source: "TGBot",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.NewInternalError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to add server source", "error", err, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", "add server source", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response AddServerSourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server source added successfully",
		"server_id", response.ServerID,
		"source", response.Source,
		"message", response.Message)

	return &response, nil
}

// ValidateServerID validates server ID format
func ValidateServerID(serverID string) error {
	if serverID == "" {
		return errors.NewValidationError("server ID cannot be empty", nil)
	}

	// Basic validation - server ID should start with "srv_" and have reasonable length
	if len(serverID) < 4 {
		return errors.NewValidationError("server ID too short", map[string]interface{}{"min_length": 4})
	}

	if len(serverID) > 100 {
		return errors.NewValidationError("server ID too long", map[string]interface{}{"max_length": 100})
	}

	return nil
}

// GetServerMetrics retrieves server metrics by server key
func (c *Client) GetServerMetrics(ctx context.Context, serverKey string) (*domain.MetricsResponse, error) {
	c.logger.Debug("Getting server metrics", "server_key", serverKey)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/unified", c.baseURL, serverKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get server metrics", "error", err, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", "get server metrics", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response domain.MetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server metrics retrieved successfully", "server_key", serverKey)

	return &response, nil
}

// AddTelegramIdentifier adds Telegram ID to server source identifiers
func (c *Client) AddTelegramIdentifier(ctx context.Context, serverKey, telegramID, username, firstName string) (*AddIdentifierResponse, error) {
	c.logger.Debug("Adding Telegram identifier", "server_key", serverKey, "telegram_id", telegramID)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources/identifiers", c.baseURL, serverKey)

	reqBody := AddIdentifierRequest{
		SourceType:     "TGBot",
		Identifiers:    []string{telegramID},
		IdentifierType: "telegram_id",
		Metadata: map[string]string{
			"chat_type":  "private",
			"username":   username,
			"first_name": firstName,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.NewInternalError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to add Telegram identifier", "error", err, "server_key", serverKey, "telegram_id", telegramID)
		return nil, errors.NewExternalError("ServerEye API", "add telegram identifier", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		// Success case
	case http.StatusNotFound:
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	case http.StatusBadRequest:
		// Could be "already exists" or "invalid data"
		body, _ := io.ReadAll(resp.Body)
		errorMsg := string(body)
		if strings.Contains(errorMsg, "already exist") {
			c.logger.Info("Telegram ID already exists", "server_key", serverKey, "telegram_id", telegramID)
			// This is not necessarily an error - we can return success
			return &AddIdentifierResponse{
				Message:        "Identifier already exists",
				ServerKey:      serverKey,
				SourceType:     "TGBot",
				Identifiers:    []string{telegramID},
				IdentifierType: "telegram_id",
			}, nil
		}
		c.logger.Error("Bad request", "server_key", serverKey, "telegram_id", telegramID, "error", errorMsg)
		return nil, errors.NewValidationError("invalid request data", map[string]interface{}{
			"server_key":  serverKey,
			"telegram_id": telegramID,
			"api_error":   errorMsg,
		})
	default:
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey, "telegram_id", telegramID)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response AddIdentifierResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Telegram identifier added successfully",
		"server_id", response.ServerID,
		"server_key", response.ServerKey,
		"telegram_id", telegramID,
		"message", response.Message)

	return &response, nil
}

// RemoveServerSource removes a source from server by key
func (c *Client) RemoveServerSource(ctx context.Context, serverKey, source string) error {
	c.logger.Debug("Removing server source", "server_key", serverKey, "source", source)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources/%s", c.baseURL, serverKey, source)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to remove server source", "error", err, "server_key", serverKey, "source", source)
		return errors.NewExternalError("ServerEye API", "remove server source", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server or source not found", "server_key", serverKey, "source", source, "status", resp.StatusCode)
		return errors.NewNotFoundError(fmt.Sprintf("server with key '%s' or source '%s'", serverKey, source))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey, "source", source)
		return errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	c.logger.Info("Server source removed successfully", "server_key", serverKey, "source", source)
	return nil
}

// RemoveServerIdentifiers removes specific identifiers from server
func (c *Client) RemoveServerIdentifiers(ctx context.Context, serverKey string, identifiers []string) error {
	c.logger.Debug("Removing server identifiers", "server_key", serverKey, "identifiers", identifiers)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources/identifiers", c.baseURL, serverKey)

	reqBody := map[string][]string{
		"identifiers": identifiers,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.NewInternalError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.NewInternalError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to remove server identifiers", "error", err, "server_key", serverKey)
		return errors.NewExternalError("ServerEye API", "remove server identifiers", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	c.logger.Info("Server identifiers removed successfully", "server_key", serverKey, "identifiers", identifiers)
	return nil
}

// GetServerStatus gets server status by server key
func (c *Client) GetServerStatus(ctx context.Context, serverKey string) (*domain.ServerStatusResponse, error) {
	c.logger.Debug("Getting server status", "server_key", serverKey)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/status", c.baseURL, serverKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get server status", "error", err, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", "get server status", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response domain.ServerStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server status retrieved successfully",
		"server_id", response.ServerID,
		"server_key", response.ServerKey,
		"online", response.Online)

	return &response, nil
}

// GetServerStaticInfo gets server static information by server key
func (c *Client) GetServerStaticInfo(ctx context.Context, serverKey string) (*domain.StaticInfoResponse, error) {
	c.logger.Debug("Getting server static info", "server_key", serverKey)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/static-info", c.baseURL, serverKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get server static info", "error", err, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", "get server static info", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_key", serverKey, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with key '%s'", serverKey))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response domain.StaticInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server static info retrieved successfully",
		"server_id", response.ServerInfo.ServerID,
		"hostname", response.ServerInfo.Hostname)

	return &response, nil
}

// RemoveServerSourceIdentifiers removes specific identifiers from a specific source
func (c *Client) RemoveServerSourceIdentifiers(ctx context.Context, serverKey, source string, identifiers []string) error {
	c.logger.Debug("Removing server source identifiers", "server_key", serverKey, "source", source, "identifiers", identifiers)

	url := fmt.Sprintf("%s/api/servers/by-key/%s/sources/%s/identifiers", c.baseURL, serverKey, source)

	reqBody := map[string][]string{
		"identifiers": identifiers,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.NewInternalError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.NewInternalError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to remove server source identifiers", "error", err, "server_key", serverKey, "source", source)
		return errors.NewExternalError("ServerEye API", "remove server source identifiers", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server or source not found", "server_key", serverKey, "source", source, "status", resp.StatusCode)
		return errors.NewNotFoundError(fmt.Sprintf("server with key '%s' or source '%s'", serverKey, source))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_key", serverKey, "source", source)
		return errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	c.logger.Info("Server source identifiers removed successfully", "server_key", serverKey, "source", source, "identifiers", identifiers)
	return nil
}
