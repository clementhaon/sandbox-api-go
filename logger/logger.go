package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"
)

// ContextKey type for context keys
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
)

// Global slog logger
var global *slog.Logger

// Initialize sets up the global logger with a JSON handler.
func Initialize() {
	level := slog.LevelInfo
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		switch strings.ToUpper(env) {
		case "DEBUG":
			level = slog.LevelDebug
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		}
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	global = slog.New(handler)
	slog.SetDefault(global)
}

func get() *slog.Logger {
	if global == nil {
		Initialize()
	}
	return global
}

// ctxAttrs extracts request_id and user_id from context as slog attributes.
func ctxAttrs(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr
	if ctx == nil {
		return attrs
	}
	if rid, ok := ctx.Value(RequestIDKey).(string); ok {
		attrs = append(attrs, slog.String("request_id", rid))
	}
	if uid, ok := ctx.Value(UserIDKey).(int); ok {
		attrs = append(attrs, slog.Int("user_id", uid))
	}
	return attrs
}

// fieldsToAttrs converts a map[string]interface{} to slog.Attr slice.
func fieldsToAttrs(fields map[string]interface{}) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

// toArgs converts slog.Attr slices to []any for slog methods.
func toArgs(attrSets ...[]slog.Attr) []any {
	var args []any
	for _, set := range attrSets {
		for _, a := range set {
			args = append(args, a)
		}
	}
	return args
}

// --- Public API (signatures preserved for compatibility) ---

func Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(context.Background(), slog.LevelDebug, message, fieldsToAttrs(f)...)
}

func DebugContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(ctx, slog.LevelDebug, message, append(ctxAttrs(ctx), fieldsToAttrs(f)...)...)
}

func Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(context.Background(), slog.LevelInfo, message, fieldsToAttrs(f)...)
}

func InfoContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(ctx, slog.LevelInfo, message, append(ctxAttrs(ctx), fieldsToAttrs(f)...)...)
}

func Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(context.Background(), slog.LevelWarn, message, fieldsToAttrs(f)...)
}

func WarnContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	get().LogAttrs(ctx, slog.LevelWarn, message, append(ctxAttrs(ctx), fieldsToAttrs(f)...)...)
}

func Error(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	attrs := fieldsToAttrs(f)
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	get().LogAttrs(context.Background(), slog.LevelError, message, attrs...)
}

func ErrorContext(ctx context.Context, message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	attrs := append(ctxAttrs(ctx), fieldsToAttrs(f)...)
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	get().LogAttrs(ctx, slog.LevelError, message, attrs...)
}

func Fatal(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	attrs := fieldsToAttrs(f)
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	get().LogAttrs(context.Background(), slog.LevelError, message, attrs...)
	os.Exit(1)
}

// LogHTTPRequest logs HTTP request details.
func LogHTTPRequest(ctx context.Context, method, url string, statusCode int, duration time.Duration) {
	attrs := append(ctxAttrs(ctx),
		slog.String("method", method),
		slog.String("url", url),
		slog.Int("status_code", statusCode),
		slog.String("duration", duration.String()),
	)
	get().LogAttrs(ctx, slog.LevelInfo, "HTTP Request", attrs...)
}

// LogDatabaseOperation logs database operation details.
func LogDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, err error) {
	attrs := append(ctxAttrs(ctx),
		slog.String("operation", operation),
		slog.String("table", table),
		slog.String("duration", duration.String()),
	)

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		get().LogAttrs(ctx, slog.LevelError, "Database operation failed", attrs...)
	} else {
		get().LogAttrs(ctx, slog.LevelInfo, "Database operation completed", attrs...)
	}
}
