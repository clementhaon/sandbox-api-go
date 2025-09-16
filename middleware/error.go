package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/metrics"
	"strings"
	"time"
)

// ErrorHandler is a custom handler type that can return errors
type ErrorHandler func(http.ResponseWriter, *http.Request) error

// ErrorMiddleware wraps handlers to provide centralized error handling
func ErrorMiddleware(handler ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add request ID to context for tracking
		requestID := generateRequestID()
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// Set request ID header for client reference
		w.Header().Set("X-Request-ID", requestID)

		// Record start time for duration logging
		startTime := time.Now()

		// Execute the handler
		err := handler(w, r)
		
		// Log the request
		duration := time.Since(startTime)
		
		if err != nil {
			handleError(w, r, err, requestID)
		}

		// Record metrics and log requests
		statusCode := 200
		if err != nil {
			if appErr, ok := errors.IsAppError(err); ok {
				statusCode = appErr.StatusCode
			} else {
				statusCode = 500
			}
		} else {
			logger.LogHTTPRequest(ctx, r.Method, r.URL.Path, statusCode, duration)
		}
		
		// Record Prometheus metrics
		endpoint := normalizeEndpoint(r.URL.Path)
		metrics.RecordHTTPRequest(r.Method, endpoint, statusCode, duration)
	}
}

// handleError processes and responds to errors
func handleError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	ctx := r.Context()
	
	// Check if it's already an AppError
	if appErr, ok := errors.IsAppError(err); ok {
		// Add request ID to the error
		appErr.WithRequestID(requestID)
		
		// Record error metrics
		metrics.RecordError(string(appErr.Type), string(appErr.Code))
		
		// Log the error with appropriate level
		if appErr.Type == errors.ErrorTypeServer {
			logger.ErrorContext(ctx, "Server error occurred", err, map[string]interface{}{
				"status_code": appErr.StatusCode,
				"error_code":  appErr.Code,
			})
		} else {
			logger.WarnContext(ctx, "Client error occurred", map[string]interface{}{
				"status_code": appErr.StatusCode,
				"error_code":  appErr.Code,
				"message":     appErr.Message,
			})
		}
		
		// Write the structured error response
		errors.WriteError(w, appErr)
		return
	}

	// Handle unexpected/unstructured errors
	metrics.RecordError("server_error", "unhandled_error")
	logger.ErrorContext(ctx, "Unhandled error occurred", err, map[string]interface{}{
		"stack_trace": string(debug.Stack()),
	})

	// Convert to internal server error
	internalErr := errors.NewInternalError().
		WithCause(err).
		WithRequestID(requestID)

	errors.WriteError(w, internalErr)
}

// PanicRecoveryMiddleware recovers from panics and converts them to errors
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				// Get request ID if it exists
				var requestID string
				if id, ok := r.Context().Value(logger.RequestIDKey).(string); ok {
					requestID = id
				} else {
					requestID = generateRequestID()
				}

				// Log the panic
				logger.ErrorContext(r.Context(), "Panic recovered", nil, map[string]interface{}{
					"panic":       recovered,
					"stack_trace": string(debug.Stack()),
					"request_id":  requestID,
				})

				// Create error response
				panicErr := errors.NewInternalError().
					WithRequestID(requestID).
					WithDetails(map[string]interface{}{
						"panic_recovered": true,
					})

				errors.WriteError(w, panicErr)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestLoggingMiddleware logs all incoming requests
func RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add request ID if not already present
		var requestID string
		if id, ok := r.Context().Value(logger.RequestIDKey).(string); ok {
			requestID = id
		} else {
			requestID = generateRequestID()
			ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
			r = r.WithContext(ctx)
		}

		// Set request ID header
		wrapper.Header().Set("X-Request-ID", requestID)

		// Execute next handler
		next.ServeHTTP(wrapper, r)

		// Log the completed request
		duration := time.Since(startTime)
		logger.LogHTTPRequest(r.Context(), r.Method, r.URL.Path, wrapper.statusCode, duration)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production you might want to use UUID
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// normalizeEndpoint normalizes URL paths for metrics (replace IDs with {id})
func normalizeEndpoint(path string) string {
	// Replace task IDs with {id} for consistent metrics
	if strings.HasPrefix(path, "/api/tasks/") && len(path) > 11 {
		return "/api/tasks/{id}"
	}
	return path
}