package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

// AddServerSource adds TGBot source to server
func (c *Client) AddServerSource(ctx context.Context, serverID string) (*AddServerSourceResponse, error) {
	c.logger.Debug("Adding server source", "server_id", serverID, "source", "TGBot")

	url := fmt.Sprintf("%s/api/servers/%s/sources", c.baseURL, serverID)

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
		c.logger.Error("Failed to add server source", "error", err, "server_id", serverID)
		return nil, errors.NewExternalError("ServerEye API", "add server source", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Warn("Server not found", "server_id", serverID, "status", resp.StatusCode)
		return nil, errors.NewNotFoundError(fmt.Sprintf("server with ID '%s'", serverID))
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code", "status", resp.StatusCode, "server_id", serverID)
		return nil, errors.NewExternalError("ServerEye API", fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	var response AddServerSourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewInternalError("failed to decode response", err)
	}

	c.logger.Info("Server source added successfully",
		"server_id", serverID,
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
