package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereyebot/pkg/protocol"
	"github.com/servereye/servereyebot/pkg/redis/streams"
	"github.com/sirupsen/logrus"
)

// sendCommandViaKafka sends command using Kafka producer and waits for response via response consumer
func (b *Bot) sendCommandViaKafka(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	if b.commandProducer == nil {
		return nil, fmt.Errorf("Kafka command producer not initialized")
	}
	if b.responseConsumer == nil {
		return nil, fmt.Errorf("Kafka response consumer not initialized")
	}

	b.logger.Debug("Sending command via Kafka",
		StringField("command_id", command.ID),
		StringField("command_type", string(command.Type)),
		StringField("server_key", serverKey),
	)

	// Send command via Kafka producer
	if err := b.commandProducer.SendCommand(ctx, serverKey, command); err != nil {
		return nil, fmt.Errorf("failed to send command via Kafka: %w", err)
	}

	// Wait for response via response consumer
	response, err := b.responseConsumer.WaitForResponse(command.ID, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response from Kafka: %w", err)
	}

	b.logger.Debug("Received response via Kafka",
		StringField("command_id", command.ID),
		StringField("response_type", string(response.Type)),
	)

	return response, nil
}

// sendCommandViaStreams sends command using PURE Streams
func (b *Bot) sendCommandViaStreams(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	// Use PURE Streams if available
	if b.streamsClient != nil {
		b.logger.Info("Sending via Streams")

		var logger *logrus.Logger
		if sl, ok := b.logger.(*StructuredLogger); ok {
			logger = sl.logger
		} else {
			logger = logrus.New()
		}

		adapter := streams.NewBotAdapter(b.streamsClient, logger)
		response, err := adapter.SendCommand(ctx, serverKey, command, timeout)
		if err == nil {
			b.logger.Info("Streams success")
			return response, nil
		}
		b.logger.Error("Streams failed", err)
	}

	// No Streams available - use Pub/Sub fallback
	b.logger.Info("Fallback to Pub/Sub")
	return b.sendCommandViaPubSub(ctx, serverKey, command, timeout)
}

// sendCommandViaPubSub is the old Pub/Sub implementation (for fallback)
func (b *Bot) sendCommandViaPubSub(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	// Create unique response channel
	responseChannel := fmt.Sprintf("resp:%s:%s", serverKey, command.ID)

	subscription, err := b.redisClient.Subscribe(ctx, responseChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	defer subscription.Close()

	// Send command
	commandChannel := fmt.Sprintf("cmd:%s", serverKey)
	messageData, err := command.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize: %w", err)
	}

	if err := b.redisClient.Publish(ctx, commandChannel, messageData); err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	// Wait for response
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for response")
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}
			return resp, nil
		}
	}
}

// getContainersViaStreams fetches containers using Streams
func (b *Bot) getContainersViaStreams(serverKey string) (*protocol.ContainersPayload, error) {
	return sendCommandAndParse[protocol.ContainersPayload](
		b,
		serverKey,
		protocol.TypeGetContainers,
		nil,
		protocol.TypeContainersResponse,
		10*time.Second,
	)
}
