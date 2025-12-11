package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/servereye/servereyebot/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// CommandProducer отправляет команды в Kafka
type CommandProducer struct {
	writer        *kafka.Writer
	logger        *logrus.Logger
	brokers       []string
	compression   string
	batchSize     int
	batchTimeout  time.Duration
}

// CommandProducerConfig конфигурация producer
type CommandProducerConfig struct {
	Brokers      []string
	Topic        string
	Compression  string
	BatchSize    int
	BatchTimeout time.Duration
}

// NewCommandProducer создает новый producer для команд
func NewCommandProducer(cfg CommandProducerConfig, logger *logrus.Logger) (*CommandProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Устанавливаем значения по умолчанию
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1 // Для команд отправляем сразу, не ждем батча
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 10 * time.Millisecond
	}
	if cfg.Compression == "" {
		cfg.Compression = "snappy"
	}

	logger.WithFields(logrus.Fields{
		"brokers":     cfg.Brokers,
		"compression": cfg.Compression,
		"batch_size":  cfg.BatchSize,
	}).Info("Kafka command producer initialized")

	// Не создаем writer здесь, так как топик будет динамическим
	return &CommandProducer{
		brokers:      cfg.Brokers,
		logger:       logger,
		compression:  cfg.Compression,
		batchSize:    cfg.BatchSize,
		batchTimeout: cfg.BatchTimeout,
	}, nil
}

// SendCommand отправляет команду в Kafka
func (p *CommandProducer) SendCommand(ctx context.Context, serverKey string, command *protocol.Message) error {
	// Сериализуем команду
	value, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Создаем динамический топик для сервера
	topic := fmt.Sprintf("cmd.%s", serverKey)

	// Создаем writer для этого топика
	writer := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    p.batchSize,
		BatchTimeout: p.batchTimeout,
		Async:        false, // Для команд используем синхронную отправку
	}

	// Устанавливаем compression
	switch p.compression {
	case "gzip":
		writer.Compression = kafka.Gzip
	case "snappy":
		writer.Compression = kafka.Snappy
	case "lz4":
		writer.Compression = kafka.Lz4
	case "zstd":
		writer.Compression = kafka.Zstd
	default:
		writer.Compression = kafka.Compression(0) // None
	}
	defer writer.Close()

	// Создаем сообщение
	msg := kafka.Message{
		Key:   []byte(serverKey), // Ключ для партиционирования по серверу
		Value: value,
		Time:  command.Timestamp,
		Headers: []kafka.Header{
			{Key: "server_key", Value: []byte(serverKey)},
			{Key: "command_id", Value: []byte(command.ID)},
			{Key: "command_type", Value: []byte(string(command.Type))},
			{Key: "server_id", Value: []byte(command.ServerID)},
		},
	}

	// Отправляем сообщение
	start := time.Now()
	err = writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"command_id":   command.ID,
			"command_type": command.Type,
			"server_key":   serverKey,
			"topic":        topic,
			"error":        err,
		}).Error("Failed to send command to Kafka")
		return fmt.Errorf("kafka write failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"server_key":   serverKey,
		"topic":        topic,
		"duration":     time.Since(start),
	}).Debug("Command sent to Kafka")

	return nil
}

// Close закрывает producer
func (p *CommandProducer) Close() error {
	p.logger.Info("Kafka command producer closed")
	return nil
}

// Stats возвращает статистику producer (заглушка, так как writer динамический)
func (p *CommandProducer) Stats() kafka.WriterStats {
	// Возвращаем пустую статистику, так как writer создается динамически для каждой команды
	return kafka.WriterStats{}
}
