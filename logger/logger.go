package logger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	FATAL LogLevel = "FATAL"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      int                    `json:"user_id,omitempty"`
	Method      string                 `json:"method,omitempty"`
	URL         string                 `json:"url,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	File        string                 `json:"file,omitempty"`
	Function    string                 `json:"function,omitempty"`
}

// Logger represents the application logger
type Logger struct {
	output   io.Writer
	minLevel LogLevel
}

// Global logger instance
var globalLogger *Logger

// ContextKey type for context keys
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
)

// Initialize sets up the global logger
func Initialize() {
	globalLogger = &Logger{
		output:   os.Stdout,
		minLevel: INFO,
	}

	// Set minimum level based on environment
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		switch strings.ToUpper(env) {
		case "DEBUG":
			globalLogger.minLevel = DEBUG
		case "INFO":
			globalLogger.minLevel = INFO
		case "WARN":
			globalLogger.minLevel = WARN
		case "ERROR":
			globalLogger.minLevel = ERROR
		case "FATAL":
			globalLogger.minLevel = FATAL
		}
	}
}

// shouldLog checks if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
		FATAL: 4,
	}
	return levels[level] >= levels[l.minLevel]
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}, ctx context.Context) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Fields:    fields,
	}

	// Add context information if available
	if ctx != nil {
		if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
			entry.RequestID = requestID
		}
		if userID, ok := ctx.Value(UserIDKey).(int); ok {
			entry.UserID = userID
		}
	}

	// Add caller information for errors and above
	if level == ERROR || level == FATAL {
		if pc, file, line, ok := runtime.Caller(3); ok {
			entry.File = formatFile(file, line)
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Function = fn.Name()
			}
		}
	}

	// Marshal and write
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to standard log
		log.Printf("Logger error: %v, Original: %s", err, message)
		return
	}

	l.output.Write(jsonData)
	l.output.Write([]byte("\n"))
}

// formatFile formats file path and line number
func formatFile(file string, line int) string {
	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1] + ":" + string(rune(line+'0'))
	}
	return file + ":" + string(rune(line+'0'))
}

// Debug logs a debug message
func Debug(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(DEBUG, message, fieldMap, nil)
}

// DebugContext logs a debug message with context
func DebugContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(DEBUG, message, fieldMap, ctx)
}

// Info logs an info message
func Info(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(INFO, message, fieldMap, nil)
}

// InfoContext logs an info message with context
func InfoContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(INFO, message, fieldMap, ctx)
}

// Warn logs a warning message
func Warn(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(WARN, message, fieldMap, nil)
}

// WarnContext logs a warning message with context
func WarnContext(ctx context.Context, message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	globalLogger.log(WARN, message, fieldMap, ctx)
}

// Error logs an error message
func Error(message string, err error, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	} else {
		fieldMap = make(map[string]interface{})
	}
	if err != nil {
		fieldMap["error"] = err.Error()
	}
	globalLogger.log(ERROR, message, fieldMap, nil)
}

// ErrorContext logs an error message with context
func ErrorContext(ctx context.Context, message string, err error, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	} else {
		fieldMap = make(map[string]interface{})
	}
	if err != nil {
		fieldMap["error"] = err.Error()
	}
	globalLogger.log(ERROR, message, fieldMap, ctx)
}

// Fatal logs a fatal message and exits
func Fatal(message string, err error, fields ...map[string]interface{}) {
	if globalLogger == nil {
		Initialize()
	}
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	} else {
		fieldMap = make(map[string]interface{})
	}
	if err != nil {
		fieldMap["error"] = err.Error()
	}
	globalLogger.log(FATAL, message, fieldMap, nil)
	os.Exit(1)
}

// LogHTTPRequest logs HTTP request details
func LogHTTPRequest(ctx context.Context, method, url string, statusCode int, duration time.Duration) {
	if globalLogger == nil {
		Initialize()
	}
	
	entry := LogEntry{
		Level:      INFO,
		Message:    "HTTP Request",
		Timestamp:  time.Now().UTC(),
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
		Duration:   duration,
	}

	// Add context information
	if ctx != nil {
		if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
			entry.RequestID = requestID
		}
		if userID, ok := ctx.Value(UserIDKey).(int); ok {
			entry.UserID = userID
		}
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Logger error: %v", err)
		return
	}

	globalLogger.output.Write(jsonData)
	globalLogger.output.Write([]byte("\n"))
}

// LogDatabaseOperation logs database operation details
func LogDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, err error) {
	if globalLogger == nil {
		Initialize()
	}

	fields := map[string]interface{}{
		"operation": operation,
		"table":     table,
		"duration":  duration.String(),
	}

	level := INFO
	message := "Database operation completed"
	
	if err != nil {
		level = ERROR
		message = "Database operation failed"
		fields["error"] = err.Error()
	}

	globalLogger.log(level, message, fields, ctx)
}