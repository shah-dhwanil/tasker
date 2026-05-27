package errors

import "fmt"

type ErrorType string
const (
	ResourceAlreadyExists ErrorType = "RESOURCE_ALREADY_EXISTS"
	ResourceNotFound      ErrorType = "RESOURCE_NOT_FOUND"
	Validation ErrorType = "VALIDATION_ERROR"
	Internal   ErrorType = "INTERNAL_ERROR"
	Unknown    ErrorType = "UNKNOWN_ERROR"
)

// AppError is a custom error type representing any error occured during the execution of the application.
type AppError struct {
	Type    ErrorType `json:"type"` // Type of the error, e.g., validation_error, unknown_error, etc.
	Title   string    `json:"title"` // A short, human-readable summary of the error.
	Message string    `json:"message"` // A detailed description of the error, providing more context and information about what went wrong.
	Resource *string   `json:"resource,omitempty"` // The resource associated with the error, if applicable. This field is optional and can be omitted if not relevant.
	Context map[string]any `json:"context,omitempty"` // Additional context about the error, if applicable.
	wrappedError error `json:"-"` // The original error that caused this AppError, if applicable. This field is not included in JSON serialization.
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s [%s]: %s",e.Type, e.Title, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.wrappedError
}

func NewUnknownError(wrappedError error, title, message string, context map[string]any) *AppError {
	return &AppError{
		Type:        Unknown,
		Title:       title,
		Message:     message,
		Context:     context,
		wrappedError: wrappedError,
	}
}