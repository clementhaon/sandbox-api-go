package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents the type of error
type ErrorCode string

const (
	// Authentication errors
	ErrAuthRequired      ErrorCode = "AUTH_REQUIRED"
	ErrInvalidToken      ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired      ErrorCode = "TOKEN_EXPIRED"
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrUserExists        ErrorCode = "USER_EXISTS"

	// Validation errors
	ErrValidationFailed  ErrorCode = "VALIDATION_FAILED"
	ErrInvalidJSON       ErrorCode = "INVALID_JSON"
	ErrMissingField      ErrorCode = "MISSING_FIELD"
	ErrInvalidFormat     ErrorCode = "INVALID_FORMAT"

	// Resource errors
	ErrNotFound          ErrorCode = "NOT_FOUND"
	ErrForbidden         ErrorCode = "FORBIDDEN"
	ErrConflict          ErrorCode = "CONFLICT"

	// Server errors
	ErrInternal          ErrorCode = "INTERNAL_ERROR"
	ErrDatabase          ErrorCode = "DATABASE_ERROR"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// Method errors
	ErrMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
)

// ErrorType categorizes errors by their nature
type ErrorType string

const (
	ErrorTypeClient    ErrorType = "client_error"     // 4xx errors
	ErrorTypeServer    ErrorType = "server_error"     // 5xx errors
	ErrorTypeValidation ErrorType = "validation_error" // Input validation errors
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode         `json:"code"`
	Message    string           `json:"message"`
	Type       ErrorType        `json:"type"`
	Details    interface{}      `json:"details,omitempty"`
	Validation []ValidationError `json:"validation,omitempty"`
	StatusCode int              `json:"-"`
	Timestamp  time.Time        `json:"timestamp"`
	RequestID  string           `json:"request_id,omitempty"`
	Cause      error            `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithCause adds a root cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithDetails adds additional details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, statusCode int, errorType ErrorType) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Type:       errorType,
		StatusCode: statusCode,
		Timestamp:  time.Now().UTC(),
	}
}

// Authentication Errors
func NewAuthRequiredError() *AppError {
	return NewAppError(ErrAuthRequired, "Authentication required", http.StatusUnauthorized, ErrorTypeClient)
}

func NewInvalidTokenError() *AppError {
	return NewAppError(ErrInvalidToken, "Invalid or malformed token", http.StatusUnauthorized, ErrorTypeClient)
}

func NewTokenExpiredError() *AppError {
	return NewAppError(ErrTokenExpired, "Token has expired", http.StatusUnauthorized, ErrorTypeClient)
}

func NewInvalidCredentialsError() *AppError {
	return NewAppError(ErrInvalidCredentials, "Invalid username or password", http.StatusUnauthorized, ErrorTypeClient)
}

func NewUserExistsError() *AppError {
	return NewAppError(ErrUserExists, "User already exists", http.StatusConflict, ErrorTypeClient)
}

// Validation Errors
func NewValidationError(validationErrors []ValidationError) *AppError {
	err := NewAppError(ErrValidationFailed, "Input validation failed", http.StatusBadRequest, ErrorTypeValidation)
	err.Validation = validationErrors
	return err
}

func NewInvalidJSONError() *AppError {
	return NewAppError(ErrInvalidJSON, "Invalid JSON format", http.StatusBadRequest, ErrorTypeClient)
}

func NewMissingFieldError(field string) *AppError {
	return NewAppError(ErrMissingField, fmt.Sprintf("Missing required field: %s", field), http.StatusBadRequest, ErrorTypeValidation)
}

func NewInvalidFormatError(field, expected string) *AppError {
	return NewAppError(ErrInvalidFormat, fmt.Sprintf("Invalid format for field '%s', expected: %s", field, expected), http.StatusBadRequest, ErrorTypeValidation)
}

// Resource Errors
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound, ErrorTypeClient)
}

func NewForbiddenError() *AppError {
	return NewAppError(ErrForbidden, "Access forbidden", http.StatusForbidden, ErrorTypeClient)
}

func NewConflictError(message string) *AppError {
	return NewAppError(ErrConflict, message, http.StatusConflict, ErrorTypeClient)
}

// Server Errors
func NewInternalError() *AppError {
	return NewAppError(ErrInternal, "Internal server error", http.StatusInternalServerError, ErrorTypeServer)
}

func NewDatabaseError() *AppError {
	return NewAppError(ErrDatabase, "Database operation failed", http.StatusInternalServerError, ErrorTypeServer)
}

func NewServiceUnavailableError() *AppError {
	return NewAppError(ErrServiceUnavailable, "Service temporarily unavailable", http.StatusServiceUnavailable, ErrorTypeServer)
}

// Method Errors
func NewMethodNotAllowedError() *AppError {
	return NewAppError(ErrMethodNotAllowed, "Method not allowed", http.StatusMethodNotAllowed, ErrorTypeClient)
}

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Error     *AppError `json:"error"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

// NewErrorResponse creates a standardized error response
func NewErrorResponse(err *AppError) *ErrorResponse {
	return &ErrorResponse{
		Error:     err,
		Success:   false,
		Timestamp: time.Now().UTC(),
	}
}

// WriteError writes an error response to the HTTP response writer
func WriteError(w http.ResponseWriter, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	
	response := NewErrorResponse(err)
	json.NewEncoder(w).Encode(response)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}