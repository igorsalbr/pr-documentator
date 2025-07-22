package logger

import (
	"github.com/igorsal/pr-documentator/internal/interfaces"
)

// Adapter adapts the existing Logger to implement interfaces.Logger
type Adapter struct {
	logger *Logger
}

// NewAdapter creates a new logger adapter
func NewAdapter(level, format string) interfaces.Logger {
	return &Adapter{
		logger: New(level, format),
	}
}

// Debug logs a debug message with optional fields
func (a *Adapter) Debug(msg string, fields ...interface{}) {
	a.logger.Debug(msg, fields...)
}

// Info logs an info message with optional fields  
func (a *Adapter) Info(msg string, fields ...interface{}) {
	a.logger.Info(msg, fields...)
}

// Warn logs a warning message with optional fields
func (a *Adapter) Warn(msg string, fields ...interface{}) {
	a.logger.Warn(msg, fields...)
}

// Error logs an error message with optional fields
func (a *Adapter) Error(msg string, err error, fields ...interface{}) {
	a.logger.Error(msg, err, fields...)
}

// Fatal logs a fatal message and exits
func (a *Adapter) Fatal(msg string, err error, fields ...interface{}) {
	a.logger.Fatal(msg, err, fields...)
}