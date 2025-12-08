package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// KafkaConsumer handles consuming metrics from Kafka
type KafkaConsumer struct {
	reader     *kafka.Reader
	redis      *redis.Client
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	topic      string
	groupID    string
}

// MetricData represents a metric from Kafka
type MetricData struct {
	ServerKey string    `json:"server_key"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(kafkaBrokers []string, topic, groupID string, redisClient *redis.Client, logger *logrus.Logger) (*KafkaConsumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaBrokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	ctx, cancel := context.WithCancel(context.Background())

	kc := &KafkaConsumer{
		reader:  reader,
		redis:   redisClient,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		topic:   topic,
		groupID: groupID,
	}

	return kc, nil
}

// Start begins consuming messages from Kafka
func (kc *KafkaConsumer) Start() error {
	kc.logger.Infof("Kafka consumer started for topic: %s, group: %s", kc.topic, kc.groupID)

	// Start consumption goroutine
	go kc.consumeLoop()

	return nil
}

// Stop stops the consumer
func (kc *KafkaConsumer) Stop() error {
	kc.cancel()
	
	if kc.reader != nil {
		kc.reader.Close()
	}

	kc.logger.Info("Kafka consumer stopped")
	return nil
}

// consumeLoop runs the main consumption loop
func (kc *KafkaConsumer) consumeLoop() {
	for {
		select {
		case <-kc.ctx.Done():
			return
		default:
			// Read message with timeout
			msg, err := kc.reader.ReadMessage(kc.ctx)
			if err != nil {
				// Check if context was cancelled
				if kc.ctx.Err() != nil {
					return
				}
				kc.logger.Errorf("Failed to read message: %v", err)
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
		ServerID   string                 `json:"server_id"`
		ServerKey  string                 `json:"server_key"`
		ServerName string                 `json:"server_name,omitempty"`
		Type       string                 `json:"type"`
		Timestamp  time.Time              `json:"timestamp"`
		Value      interface{}            `json:"value"`
		Tags       map[string]string      `json:"tags,omitempty"`
		Version    string                 `json:"version"`
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
		if err := kc.cacheMetric(metric); err != nil {
			kc.logger.Errorf("Failed to cache metric: %v", err)
		}
		
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
				if err := kc.cacheMetric(metric); err != nil {
					kc.logger.Errorf("Failed to cache metric: %v", err)
				}
			}
		}
		
	default:
		kc.logger.Warnf("Unsupported metric value type: %T", v)
	}
}

// cacheMetric stores the metric in Redis for fast access
func (kc *KafkaConsumer) cacheMetric(metric MetricData) error {
	// Use a hash for each server's metrics
	hashKey := fmt.Sprintf("metrics:%s", metric.ServerKey)
	
	// Store individual metric
	field := fmt.Sprintf("%s:%s", metric.Name, metric.Unit)
	value := fmt.Sprintf("%.2f", metric.Value)
	
	// Set the metric value with expiration (e.g., 5 minutes)
	pipe := kc.redis.Pipeline()
	pipe.HSet(kc.ctx, hashKey, field, value)
	pipe.HSet(kc.ctx, hashKey, fmt.Sprintf("%s:timestamp", field), metric.Timestamp.Format(time.RFC3339))
	pipe.Expire(kc.ctx, hashKey, 5*time.Minute)
	
	// Also store in a sorted set for latest metrics per server type
	zKey := fmt.Sprintf("latest_metrics:%s", metric.Name)
	zScore := float64(metric.Timestamp.Unix())
	zMember := fmt.Sprintf("%s:%s", metric.ServerKey, metric.Unit)
	pipe.ZAdd(kc.ctx, zKey, redis.Z{Score: zScore, Member: zMember})
	pipe.Expire(kc.ctx, zKey, 24*time.Hour)
	
	// Keep only last 100 entries per metric type
	pipe.ZRemRangeByRank(kc.ctx, zKey, 0, -101)
	
	_, err := pipe.Exec(kc.ctx)
	return err
}

// GetCachedMetric retrieves a cached metric from Redis
func (kc *KafkaConsumer) GetCachedMetric(serverKey, metricName, unit string) (float64, *time.Time, error) {
	hashKey := fmt.Sprintf("metrics:%s", serverKey)
	field := fmt.Sprintf("%s:%s", metricName, unit)
	
	pipe := kc.redis.Pipeline()
	valueCmd := pipe.HGet(kc.ctx, hashKey, field)
	timestampCmd := pipe.HGet(kc.ctx, hashKey, fmt.Sprintf("%s:timestamp", field))
	
	_, err := pipe.Exec(kc.ctx)
	if err != nil {
		return 0, nil, err
	}
	
	value, err := valueCmd.Float64()
	if err != nil {
		return 0, nil, err
	}
	
	timestampStr, err := timestampCmd.Result()
	if err != nil {
		return 0, nil, err
	}
	
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return 0, nil, err
	}
	
	return value, &timestamp, nil
}

// GetAllCachedMetrics retrieves all cached metrics for a server
func (kc *KafkaConsumer) GetAllCachedMetrics(serverKey string) (map[string]float64, error) {
	hashKey := fmt.Sprintf("metrics:%s", serverKey)
	
	// Get all metrics (excluding timestamps)
	data, err := kc.redis.HGetAll(kc.ctx, hashKey).Result()
	if err != nil {
		return nil, err
	}
	
	metrics := make(map[string]float64)
	for key, value := range data {
		// Skip timestamp fields
		if len(key) > 9 && key[len(key)-9:] == ":timestamp" {
			continue
		}
		
		if floatValue, err := parseFloat64(value); err == nil {
			metrics[key] = floatValue
		}
	}
	
	return metrics, nil
}

// Helper functions
func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
