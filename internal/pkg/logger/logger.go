package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
	LoggerKey  contextKey = "ctx_logger"
)

// InitLogger configures global Zerolog logger based on environment.
func InitLogger(appEnv string) {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var output io.Writer = os.Stdout
	if appEnv != "production" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05.000",
		}
	}

	log.Logger = zerolog.New(output).With().
		Timestamp().
		Str("service", "payment-oms-service").
		Logger()
}

// WithTraceContext injects a child logger containing the trace_id into context.
func WithTraceContext(ctx context.Context, traceID string) context.Context {
	subLogger := log.With().Str("trace_id", traceID).Logger()
	ctx = context.WithValue(ctx, TraceIDKey, traceID)
	return context.WithValue(ctx, LoggerKey, subLogger)
}

// FromContext extracts the contextual logger or falls back to the global logger.
func FromContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return log.Logger
	}
	if l, ok := ctx.Value(LoggerKey).(zerolog.Logger); ok {
		return l
	}
	return log.Logger
}

// GetTraceID extracts trace_id from context if available.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

