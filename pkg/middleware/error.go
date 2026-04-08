package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/metrics"
)

type ErrorHandler func(http.ResponseWriter, *http.Request) error

func ErrorMiddleware(handler ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		startTime := time.Now()
		err := handler(w, r)
		duration := time.Since(startTime)

		if err != nil {
			handleError(w, r, err, requestID)
		}

		statusCode := 200
		if err != nil {
			if appErr, ok := errors.IsAppError(err); ok {
				statusCode = appErr.StatusCode
			} else {
				statusCode = 500
			}
		}

		endpoint := normalizeEndpoint(r.URL.Path)
		metrics.RecordHTTPRequest(r.Method, endpoint, statusCode, duration)
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	ctx := r.Context()

	if appErr, ok := errors.IsAppError(err); ok {
		appErr.WithRequestID(requestID)
		metrics.RecordError(string(appErr.Type), string(appErr.Code))

		if appErr.Type == errors.ErrorTypeServer {
			logger.ErrorContext(ctx, "Server error occurred", err, map[string]interface{}{
				"status_code": appErr.StatusCode, "error_code": appErr.Code,
			})
		} else {
			logger.WarnContext(ctx, "Client error occurred", map[string]interface{}{
				"status_code": appErr.StatusCode, "error_code": appErr.Code, "message": appErr.Message,
			})
		}

		errors.WriteError(w, appErr)
		return
	}

	metrics.RecordError("server_error", "unhandled_error")
	logger.ErrorContext(ctx, "Unhandled error occurred", err, map[string]interface{}{
		"stack_trace": string(debug.Stack()),
	})

	internalErr := errors.NewInternalError().WithCause(err).WithRequestID(requestID)
	errors.WriteError(w, internalErr)
}

func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				var requestID string
				if id, ok := r.Context().Value(logger.RequestIDKey).(string); ok {
					requestID = id
				} else {
					requestID = generateRequestID()
				}

				logger.ErrorContext(r.Context(), "Panic recovered", nil, map[string]interface{}{
					"panic": recovered, "stack_trace": string(debug.Stack()), "request_id": requestID,
				})

				panicErr := errors.NewInternalError().WithRequestID(requestID).WithDetails(map[string]interface{}{
					"panic_recovered": true,
				})
				errors.WriteError(w, panicErr)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		var requestID string
		if id, ok := r.Context().Value(logger.RequestIDKey).(string); ok {
			requestID = id
		} else {
			requestID = generateRequestID()
			ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
			r = r.WithContext(ctx)
		}

		wrapper.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(wrapper, r)

		if r.URL.Path != "/metrics" {
			duration := time.Since(startTime)
			logger.LogHTTPRequest(r.Context(), r.Method, r.URL.Path, wrapper.statusCode, duration)
		}
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return time.Now().Format("20060102150405") + "-" + hex.EncodeToString(b)
}

func normalizeEndpoint(path string) string {
	if strings.HasPrefix(path, "/api/tasks/") && len(path) > 11 {
		return "/api/tasks/{id}"
	}
	return path
}
