package errors

import "fmt"

type ErrorType string
const (
	Validation ErrorType = "validation_error"
	Unknown    ErrorType = "unknown_error"
)

// AppError is a custom error type representing any error occured during the execution of the application.
type appError struct {
	Type    ErrorType `json:"type"` // Type of the error, e.g., validation_error, unknown_error, etc.
	Title   string    `json:"title"` // A short, human-readable summary of the error.
	Message string    `json:"message"` // A detailed description of the error, providing more context and information about what went wrong.
	Resource *string   `json:"resource,omitempty"` // The resource associated with the error, if applicable. This field is optional and can be omitted if not relevant.
	Context map[string]any `json:"context,omitempty"` // Additional context about the error, if applicable.
	wrappedError error `json:"-"` // The original error that caused this AppError, if applicable. This field is not included in JSON serialization.
}

func (e *appError) Error() string {
	return fmt.Sprintf("%s [%s]: %s",e.Type, e.Title, e.Message)
}

func (e *appError) Unwrap() error {
	return e.wrappedError
}

func NewInternalError(title, message string, context map[string]any, wrappedError error) *appError {
	return &appError{
		Type:        Unknown,
		Title:       title,
		Message:     message,
		Context:     context,
		wrappedError: wrappedError,
	}
}