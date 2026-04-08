package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ErrorCode string

const (
	ErrAuthRequired       ErrorCode = "AUTH_REQUIRED"
	ErrInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrUserExists         ErrorCode = "USER_EXISTS"

	ErrValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrInvalidJSON      ErrorCode = "INVALID_JSON"
	ErrMissingField     ErrorCode = "MISSING_FIELD"
	ErrInvalidFormat    ErrorCode = "INVALID_FORMAT"

	ErrNotFound  ErrorCode = "NOT_FOUND"
	ErrForbidden ErrorCode = "FORBIDDEN"
	ErrConflict  ErrorCode = "CONFLICT"

	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrDatabase           ErrorCode = "DATABASE_ERROR"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	ErrMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"
)

type ErrorType string

const (
	ErrorTypeClient     ErrorType = "client_error"
	ErrorTypeServer     ErrorType = "server_error"
	ErrorTypeValidation ErrorType = "validation_error"
)

type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

type AppError struct {
	Code       ErrorCode         `json:"code"`
	Message    string            `json:"message"`
	Type       ErrorType         `json:"type"`
	Details    interface{}       `json:"details,omitempty"`
	Validation []ValidationError `json:"validation,omitempty"`
	StatusCode int               `json:"-"`
	Timestamp  time.Time         `json:"timestamp"`
	RequestID  string            `json:"request_id,omitempty"`
	Cause      error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) WithCause(cause error) *AppError    { e.Cause = cause; return e }
func (e *AppError) WithRequestID(id string) *AppError   { e.RequestID = id; return e }
func (e *AppError) WithDetails(d interface{}) *AppError  { e.Details = d; return e }

func NewAppError(code ErrorCode, message string, statusCode int, errorType ErrorType) *AppError {
	return &AppError{Code: code, Message: message, Type: errorType, StatusCode: statusCode, Timestamp: time.Now().UTC()}
}

func NewAuthRequiredError() *AppError {
	return NewAppError(ErrAuthRequired, "Authentication required", http.StatusUnauthorized, ErrorTypeClient)
}
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewAppError(ErrAuthRequired, message, http.StatusUnauthorized, ErrorTypeClient)
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

func NewValidationError(validationErrors []ValidationError) *AppError {
	err := NewAppError(ErrValidationFailed, "Input validation failed", http.StatusBadRequest, ErrorTypeValidation)
	err.Validation = validationErrors
	return err
}
func NewBadRequestError(message string) *AppError {
	if message == "" {
		message = "Bad request"
	}
	return NewAppError(ErrValidationFailed, message, http.StatusBadRequest, ErrorTypeClient)
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

func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound, ErrorTypeClient)
}
func NewForbiddenError() *AppError {
	return NewAppError(ErrForbidden, "Access forbidden", http.StatusForbidden, ErrorTypeClient)
}
func NewConflictError(message string) *AppError {
	return NewAppError(ErrConflict, message, http.StatusConflict, ErrorTypeClient)
}

func NewInternalError() *AppError {
	return NewAppError(ErrInternal, "Internal server error", http.StatusInternalServerError, ErrorTypeServer)
}
func NewInternalServerError(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return NewAppError(ErrInternal, message, http.StatusInternalServerError, ErrorTypeServer)
}
func NewDatabaseError() *AppError {
	return NewAppError(ErrDatabase, "Database operation failed", http.StatusInternalServerError, ErrorTypeServer)
}
func NewServiceUnavailableError() *AppError {
	return NewAppError(ErrServiceUnavailable, "Service temporarily unavailable", http.StatusServiceUnavailable, ErrorTypeServer)
}
func NewMethodNotAllowedError() *AppError {
	return NewAppError(ErrMethodNotAllowed, "Method not allowed", http.StatusMethodNotAllowed, ErrorTypeClient)
}

type ErrorResponse struct {
	Error     *AppError `json:"error"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

func NewErrorResponse(err *AppError) *ErrorResponse {
	return &ErrorResponse{Error: err, Success: false, Timestamp: time.Now().UTC()}
}

func WriteError(w http.ResponseWriter, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	json.NewEncoder(w).Encode(NewErrorResponse(err))
}

func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}
