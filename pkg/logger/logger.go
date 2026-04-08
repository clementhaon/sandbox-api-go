package logger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	FATAL LogLevel = "FATAL"
)

type LogEntry struct {
	Level      LogLevel               `json:"level"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     int                    `json:"user_id,omitempty"`
	Method     string                 `json:"method,omitempty"`
	URL        string                 `json:"url,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	File       string                 `json:"file,omitempty"`
	Function   string                 `json:"function,omitempty"`
}

type Logger struct {
	output   io.Writer
	minLevel LogLevel
}

var globalLogger *Logger

type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
)

func Initialize() {
	globalLogger = &Logger{
		output:   os.Stdout,
		minLevel: INFO,
	}

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

func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3, FATAL: 4,
	}
	return levels[level] >= levels[l.minLevel]
}

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

	if ctx != nil {
		if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
			entry.RequestID = requestID
		}
		if userID, ok := ctx.Value(UserIDKey).(int); ok {
			entry.UserID = userID
		}
	}

	if level == ERROR || level == FATAL {
		if pc, file, line, ok := runtime.Caller(3); ok {
			entry.File = formatFile(file, line)
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Function = fn.Name()
			}
		}
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Logger error: %v, Original: %s", err, message)
		return
	}

	l.output.Write(jsonData)
	l.output.Write([]byte("\n"))
}

func formatFile(file string, line int) string {
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1] + ":" + strconv.Itoa(line)
	}
	return file + ":" + strconv.Itoa(line)
}

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
