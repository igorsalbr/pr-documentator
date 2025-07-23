package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger instance
func New(level, format string) *Logger {
	// Parse log level
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	var logger zerolog.Logger

	// Configure output format
	if format == "console" {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	event := l.logger.Error().Err(err)
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *Logger) Fatal(msg string, err error, fields ...interface{}) {
	event := l.logger.Fatal().Err(err)
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *Logger) addFields(event *zerolog.Event, fields ...interface{}) {
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			event.Interface(key, value)
		}
	}
}

func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
