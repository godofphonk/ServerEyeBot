package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Known metric topics that the agent publishes to
var metricTopics = []string{
	"metrics.cpu",
	"metrics.memory",
	"metrics.disk",
	"metrics.uptime",
	"metrics.load",
	"metrics.network",
}

// KafkaConsumer handles consuming metrics from Kafka
type KafkaConsumer struct {
	readers []*kafka.Reader // Multiple readers for different topics
	logger  *logrus.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	groupID string
}

// MetricData represents a metric from Kafka
type MetricData struct {
	ServerKey string    `json:"server_key"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// NewKafkaConsumer creates a new Kafka consumer that can handle multiple metrics topics
func NewKafkaConsumer(kafkaBrokers []string, topicPrefix, groupID string, logger *logrus.Logger) (*KafkaConsumer, error) {
	if len(kafkaBrokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}
	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create readers for all metric topics
	var readers []*kafka.Reader
	for _, topic := range metricTopics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        kafkaBrokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       10e3, // 10KB
			MaxBytes:       10e6, // 10MB
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		})
		readers = append(readers, reader)

		logger.WithField("topic", topic).Debug("Created Kafka reader for metric topic")
	}

	kc := &KafkaConsumer{
		readers: readers,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		groupID: groupID,
	}

	logger.WithFields(logrus.Fields{
		"topics_count": len(readers),
		"group_id":     groupID,
	}).Info("Kafka multi-topic metrics consumer initialized")

	return kc, nil
}

// Start begins consuming messages from Kafka
func (kc *KafkaConsumer) Start() error {
	kc.logger.Infof("Kafka consumer started for %d topics, group: %s", len(kc.readers), kc.groupID)

	// Start consumption goroutine for each reader
	for i, reader := range kc.readers {
		go kc.consumeLoopForReader(reader, metricTopics[i])
	}

	return nil
}

// Stop stops the consumer
func (kc *KafkaConsumer) Stop() error {
	kc.cancel()

	for _, reader := range kc.readers {
		if err := reader.Close(); err != nil {
			kc.logger.WithError(err).Error("Error closing Kafka reader")
		}
	}

	kc.logger.Info("Kafka consumer stopped")
	return nil
}

// consumeLoopForReader runs the main consumption loop for a specific reader
func (kc *KafkaConsumer) consumeLoopForReader(reader *kafka.Reader, topic string) {
	for {
		select {
		case <-kc.ctx.Done():
			return
		default:
			// Read message with timeout
			msg, err := reader.ReadMessage(kc.ctx)
			if err != nil {
				// Check if context was cancelled
				if kc.ctx.Err() != nil {
					return
				}
				kc.logger.WithError(err).WithField("topic", topic).Error("Failed to read message")
				continue
			}

			kc.handleMessage(&msg)
		}
	}
}

// handleMessage processes a single Kafka message
func (kc *KafkaConsumer) handleMessage(msg *kafka.Message) {
	// Parse the metric from agent format
	var agentMetric struct {
		ServerID   string            `json:"server_id"`
		ServerKey  string            `json:"server_key"`
		ServerName string            `json:"server_name,omitempty"`
		Type       string            `json:"type"`
		Timestamp  time.Time         `json:"timestamp"`
		Value      interface{}       `json:"value"`
		Tags       map[string]string `json:"tags,omitempty"`
		Version    string            `json:"version"`
	}

	if err := json.Unmarshal(msg.Value, &agentMetric); err != nil {
		kc.logger.Errorf("Failed to unmarshal metric: %v", err)
		return
	}

	// Handle different value types
	switch v := agentMetric.Value.(type) {
	case float64:
		// Simple metric like CPU temperature
		metric := MetricData{
			ServerKey: agentMetric.ServerKey,
			Name:      agentMetric.Type,
			Value:     v,
			Unit:      "", // Unit not provided in agent format
			Timestamp: agentMetric.Timestamp,
		}
		_ = metric // Metric created but not used (Redis cache removed)

	case map[string]interface{}:
		// Complex metric like memory info
		for key, value := range v {
			if floatVal, ok := value.(float64); ok {
				metric := MetricData{
					ServerKey: agentMetric.ServerKey,
					Name:      key, // Use individual field names
					Value:     floatVal,
					Unit:      "", // Unit could be inferred from key name
					Timestamp: agentMetric.Timestamp,
				}
				_ = metric // Metric created but not used (Redis cache removed)
			}
		}

	default:
		kc.logger.Warnf("Unsupported metric value type: %T", v)
	}
}

// Helper functions
func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
