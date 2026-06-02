package errors

func NewUnauthorizedError(err error,message string, resource *string, context map[string]any) *AppError {
	return &AppError{
		Type:     Unauthorized,
		Title:    "Unauthorized",
		Message:  message,
		Resource: resource,
		Context:  context,
		wrappedError: err,
	}
}