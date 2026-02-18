// Package logger provides a structured logging interface for Kube-Reflex.
// Implementations use slog (Go 1.21+) for consistent, structured error reporting.
package logger

import (
	"log/slog"
	"os"
)

// Logger defines the structured logging contract for error reporting and observability.
// Use Error for critical failures (catch & report), Warn for non-critical issues (catch & continue).
type Logger interface {
	// Error logs critical failures. Use when the operation cannot continue or data may be lost.
	Error(msg string, err error, fields map[string]interface{})
	// Warn logs non-critical issues. Use when the operation continues but something went wrong.
	Warn(msg string, fields map[string]interface{})
	// Info logs standard informational messages.
	Info(msg string, fields map[string]interface{})
}

// SlogLogger implements Logger using the standard library slog.
type SlogLogger struct {
	log *slog.Logger
}

// NewSlogLogger returns a Logger backed by slog with JSON output for production.
func NewSlogLogger() *SlogLogger {
	return &SlogLogger{
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// NewSlogLoggerDebug returns a Logger with debug level enabled.
func NewSlogLoggerDebug() *SlogLogger {
	return &SlogLogger{
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}
}

// Error implements Logger.Error.
func (l *SlogLogger) Error(msg string, err error, fields map[string]interface{}) {
	attrs := toAttrs(fields)
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	l.log.Error(msg, attrs...)
}

// Warn implements Logger.Warn.
func (l *SlogLogger) Warn(msg string, fields map[string]interface{}) {
	l.log.Warn(msg, toAttrs(fields)...)
}

// Info implements Logger.Info.
func (l *SlogLogger) Info(msg string, fields map[string]interface{}) {
	l.log.Info(msg, toAttrs(fields)...)
}

func toAttrs(fields map[string]interface{}) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}
