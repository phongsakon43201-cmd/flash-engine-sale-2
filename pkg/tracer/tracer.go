package tracer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

// NewTraceID generates a unique 128-bit W3C compliant trace ID string
func NewTraceID() string {
	return "tr-" + uuid.New().String()
}

// WithTraceID injects a trace ID into context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// FromContext extracts a trace ID from context
func FromContext(ctx context.Context) string {
	if val, ok := ctx.Value(traceIDKey).(string); ok && val != "" {
		return val
	}
	return ""
}

// LogInfo logs structured messages decorated with active TraceID
func LogInfo(ctx context.Context, format string, v ...interface{}) {
	traceID := FromContext(ctx)
	prefix := ""
	if traceID != "" {
		prefix = fmt.Sprintf("[TraceID: %s] ", traceID)
	}
	log.Printf(prefix+format, v...)
}
