package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error codes
type ErrorCode string

const (
	// Validation errors
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeRequired     ErrorCode = "REQUIRED_FIELD"
	ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// Authentication/Authorization errors
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"

	// Business logic errors
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeConflict      ErrorCode = "CONFLICT"
	ErrCodeLimitExceeded ErrorCode = "LIMIT_EXCEEDED"

	// System errors
	ErrCodeInternal    ErrorCode = "INTERNAL_ERROR"
	ErrCodeExternal    ErrorCode = "EXTERNAL_ERROR"
	ErrCodeTimeout     ErrorCode = "TIMEOUT"
	ErrCodeUnavailable ErrorCode = "UNAVAILABLE"

	// Telegram specific errors
	ErrCodeTelegramAPI ErrorCode = "TELEGRAM_API_ERROR"
	ErrCodeRateLimit   ErrorCode = "RATE_LIMIT"
	ErrCodeBotBlocked  ErrorCode = "BOT_BLOCKED"

	// Metrics errors
	ErrCodeMetricsUnavailable ErrorCode = "METRICS_UNAVAILABLE"
	ErrCodePermissionDenied   ErrorCode = "PERMISSION_DENIED"
)

// AppError represents application error with context
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Cause      error                  `json:"-"`
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// NewRequiredFieldError creates a new required field error
func NewRequiredFieldError(field string) *AppError {
	return &AppError{
		Code:       ErrCodeRequired,
		Message:    fmt.Sprintf("Field '%s' is required", field),
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]interface{}{"field": field},
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"resource": resource},
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// NewExternalError creates a new external service error
func NewExternalError(service, message string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeExternal,
		Message:    fmt.Sprintf("%s error: %s", service, message),
		HTTPStatus: http.StatusBadGateway,
		Cause:      cause,
		Details:    map[string]interface{}{"service": service},
	}
}

// NewTelegramAPIError creates a new Telegram API error
func NewTelegramAPIError(message string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeTelegramAPI,
		Message:    fmt.Sprintf("Telegram API error: %s", message),
		HTTPStatus: http.StatusBadGateway,
		Cause:      cause,
	}
}

// NewMetricsUnavailableError creates a new metrics unavailable error
func NewMetricsUnavailableError(metric string, cause error) *AppError {
	return &AppError{
		Code:       ErrCodeMetricsUnavailable,
		Message:    fmt.Sprintf("Metric '%s' unavailable", metric),
		HTTPStatus: http.StatusServiceUnavailable,
		Cause:      cause,
		Details:    map[string]interface{}{"metric": metric},
	}
}

// IsErrorCode checks if error matches specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode extracts error code from error
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternal
}
