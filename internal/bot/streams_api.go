package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/servereye/servereyebot/pkg/protocol"
)

// sendCommandViaKafka sends command using Kafka producer and waits for response via response consumer
func (b *Bot) sendCommandViaKafka(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	if b.commandProducer == nil {
		return nil, fmt.Errorf("kafka command producer not initialized")
	}
	if b.responseConsumer == nil {
		return nil, fmt.Errorf("kafka response consumer not initialized")
	}

	b.logger.Info("KAFKA: Sending command",
		StringField("command_id", command.ID),
		StringField("command_type", string(command.Type)),
		StringField("server_key", serverKey),
		StringField("topic", fmt.Sprintf("cmd.%s", serverKey)),
	)

	// Send command via Kafka producer
	if err := b.commandProducer.SendCommand(ctx, serverKey, command); err != nil {
		b.logger.Error("KAFKA: Failed to send command", err,
			StringField("command_id", command.ID),
			StringField("server_key", serverKey),
		)
		return nil, fmt.Errorf("failed to send command via Kafka: %w", err)
	}

	b.logger.Info("KAFKA: Command sent successfully",
		StringField("command_id", command.ID),
		StringField("server_key", serverKey),
	)

	// Wait for response via response consumer
	response, err := b.responseConsumer.WaitForResponse(ctx, command.ID, timeout)
	if err != nil {
		b.logger.Error("KAFKA: Failed to receive response", err,
			StringField("command_id", command.ID),
			StringField("server_key", serverKey),
		)
		return nil, fmt.Errorf("failed to receive response via Kafka: %w", err)
	}

	b.logger.Info("KAFKA: Response received successfully",
		StringField("command_id", command.ID),
		StringField("server_key", serverKey),
		StringField("response_type", string(response.Type)),
	)

	return response, nil
}

// sendCommandAndParse sends command via HTTP API and parses response into specified payload type
func (b *Bot) sendCommandAndParse(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration, payload interface{}) error {
	// Defensive check
	if b == nil {
		return fmt.Errorf("bot instance is nil")
	}

	// Send command via HTTP API only (worldwide deployment)
	response, err := b.sendCommandViaHTTP(ctx, serverKey, command, timeout)
	if err != nil {
		return fmt.Errorf("failed to send command via HTTP: %w", err)
	}

	// Defensive check for response
	if response == nil {
		return fmt.Errorf("received nil response")
	}

	// Parse the response payload into the provided struct
	if payload != nil {
		if response.Payload == nil {
			return fmt.Errorf("response payload is nil")
		}

		payloadBytes, err := json.Marshal(response.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal response payload: %w", err)
		}

		if err := json.Unmarshal(payloadBytes, payload); err != nil {
			return fmt.Errorf("failed to unmarshal response payload: %w", err)
		}
	}

	return nil
}

func (b *Bot) sendCommandViaStreams(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	return nil, fmt.Errorf("streams transport is disabled; kafka must be enabled")
}

// sendCommandViaHTTP sends command using backend HTTP API
func (b *Bot) sendCommandViaHTTP(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	// Prepare command request
	cmdRequest := map[string]interface{}{
		"server_key": serverKey,
		"command":    string(command.Type),
		"payload":    command.Payload,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(cmdRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Create HTTP request with timeout
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Send command to backend API - POST to create command
	req, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("BACKEND_URL")+"/v1/commands/create", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response - backend returns response directly (it blocks until agent responds)
	var apiResponse struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResponse.Success {
		return nil, fmt.Errorf("API returned error: %v", apiResponse.Data)
	}

	// Convert backend response to protocol.Message
	return &protocol.Message{
		ID:      command.ID,
		Type:    command.Type,
		Payload: apiResponse.Data,
	}, nil
}

// waitForHTTPResponse polls for command response via backend API
func (b *Bot) waitForHTTPResponse(ctx context.Context, commandID string, timeout time.Duration) (*protocol.Message, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Poll for response
	pollInterval := 1 * time.Second
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("timeout waiting for response")
		case <-ticker.C:
			// Check for response
			req, err := http.NewRequestWithContext(pollCtx, "GET", fmt.Sprintf("%s/v1/commands/response/%s", os.Getenv("BACKEND_URL"), commandID), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				continue // Try again on next poll
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil || resp.StatusCode != http.StatusOK {
				continue // Try again on next poll
			}

			// Parse response
			var apiResponse struct {
				Success bool        `json:"success"`
				Data    interface{} `json:"data"`
			}

			if err := json.Unmarshal(body, &apiResponse); err != nil {
				continue // Try again on next poll
			}

			if apiResponse.Success && apiResponse.Data != nil {
				// Convert response data to protocol.Message
				responseData, err := json.Marshal(apiResponse.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal response: %w", err)
				}

				var response protocol.Message
				if err := json.Unmarshal(responseData, &response); err != nil {
					return nil, fmt.Errorf("failed to unmarshal response: %w", err)
				}

				return &response, nil
			}
		}
	}
}
