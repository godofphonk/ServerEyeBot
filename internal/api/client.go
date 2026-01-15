package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	url := fmt.Sprintf("%s/api/servers/by-key/%s/metrics", c.baseURL, serverKey)

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
