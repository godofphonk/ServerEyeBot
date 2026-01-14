package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger interface defines logging methods
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
}

// LogrusLogger implements Logger interface using logrus
type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// New creates a new logger instance
func New(config LoggerConfig) (Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Set output
	switch config.Output {
	case "file":
		if config.Filename == "" {
			config.Filename = "app.log"
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(config.Filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		// Use lumberjack for log rotation
		logger.SetOutput(&lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		})
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		logger.SetOutput(os.Stdout)
	}

	return &LogrusLogger{
		logger: logger,
		entry:  logger.WithFields(logrus.Fields{}),
	}, nil
}

// Debug logs a debug message
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Debugf logs a debug message with formatting
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Info logs an info message
func (l *LogrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Infof logs an info message with formatting
func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warn logs a warning message
func (l *LogrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Warnf logs a warning message with formatting
func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Error logs an error message
func (l *LogrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Errorf logs an error message with formatting
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Fatalf logs a fatal message with formatting and exits
func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// WithField returns a logger with the specified field
func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields returns a logger with the specified fields
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}

// WithError returns a logger with the error field
func (l *LogrusLogger) WithError(err error) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithError(err),
	}
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}
