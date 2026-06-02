package errors

import "fmt"


func NewCategoryAlreadyExistsError(wrappedError error, fieldName, fieldValue string, message *string) *AppError {
	if message == nil {
		msg := fmt.Sprintf("Category with the specified %s already exists.", fieldName)
		message = &msg
	}
	return &AppError{
		Type:        ResourceAlreadyExists,
		Title:       "Category Already Exists",
		Message:     *message,
		Context:     map[string]any{fieldName: fieldValue},
		wrappedError: wrappedError,
	}
}

func NewCategoryNotFoundError(wrappedError error, message *string) *AppError {
	if message == nil {
		msg := "No category found with the specified data"
		message = &msg
	}
	return &AppError{
		Type:        ResourceNotFound,
		Title:       "Category Not Found",
		Message:     *message,
		wrappedError: wrappedError,
	}
}