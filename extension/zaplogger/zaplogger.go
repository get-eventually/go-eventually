package zaplogger

import (
	"go.uber.org/zap"

	"github.com/get-eventually/go-eventually/logger"
)

var _ logger.Logger = &Logger{}

// Logger is a zap wrapper that implements the eventually logger.Logger interface.
type Logger zap.Logger

func adaptFields(fields []logger.Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))

	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}

	return zapFields
}

// Debug prints a debug log message.
func (l *Logger) Debug(msg string, fields ...logger.Field) {
	(*zap.Logger)(l).Debug(msg, adaptFields(fields)...)
}

// Info prints an info log message.
func (l *Logger) Info(msg string, fields ...logger.Field) {
	(*zap.Logger)(l).Info(msg, adaptFields(fields)...)
}

// Error prints an error log message.
func (l *Logger) Error(msg string, fields ...logger.Field) {
	(*zap.Logger)(l).Error(msg, adaptFields(fields)...)
}

// Wrap wraps a zap.Logger into a zaplogger.Logger instance.
func Wrap(l *zap.Logger) *Logger {
	return (*Logger)(l)
}
