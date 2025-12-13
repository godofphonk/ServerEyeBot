package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/servereye/servereyebot/pkg/protocol"
)

// sendCommandViaKafka sends command using Kafka producer and waits for response via response consumer
func (b *Bot) sendCommandViaKafka(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	if b.commandProducer == nil {
		return nil, fmt.Errorf("Kafka command producer not initialized")
	}
	if b.responseConsumer == nil {
		return nil, fmt.Errorf("Kafka response consumer not initialized")
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

// sendCommandAndParse sends command via Kafka and parses response into specified payload type
func (b *Bot) sendCommandAndParse(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration, payload interface{}) error {
	// Defensive check
	if b == nil {
		return fmt.Errorf("bot instance is nil")
	}

	response, err := b.sendCommandViaKafka(ctx, serverKey, command, timeout)
	if err != nil {
		return fmt.Errorf("failed to send command via Kafka: %w", err)
	}

	// Defensive check for response
	if response == nil {
		return fmt.Errorf("received nil response from Kafka")
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
