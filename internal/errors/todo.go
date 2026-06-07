package errors

func NewTodoNotFoundError(wrappedError error, message *string) *AppError {
	if message == nil {
		msg := "No todo found with the specified data"
		message = &msg
	}
	return &AppError{
		Type:         ResourceNotFound,
		Title:        "Todo Not Found",
		Message:      *message,
		wrappedError: wrappedError,
	}
}

func NewTodoAlreadyExistsError(wrappedError error, message *string) *AppError {
	if message == nil {
		msg := "Todo with the specified data already exists."
		message = &msg
	}
	return &AppError{
		Type:         ResourceAlreadyExists,
		Title:        "Todo Already Exists",
		Message:      *message,
		wrappedError: wrappedError,
	}
}

func NewTodoCategoryNotFoundError(wrappedError error) *AppError {
	msg := "The specified category does not exist."
	return &AppError{
		Type:         Validation,
		Title:        "Invalid Todo Category",
		Message:      msg,
		wrappedError: wrappedError,
	}
}

func NewTodoParentNotFoundError(wrappedError error) *AppError {
	msg := "The parent todo does not exist."
	return &AppError{
		Type:         Validation,
		Title:        "Invalid Todo Parent",
		Message:      msg,
		wrappedError: wrappedError,
	}
}

func NewInvalidTodoPriorityError(wrappedError error, priority int) *AppError {
	msg := "Priority must be between 1 and 5."
	return &AppError{
		Type:         Validation,
		Title:        "Invalid Todo Priority",
		Message:      msg,
		Context:      map[string]any{"priority": priority},
		wrappedError: wrappedError,
	}
}

func NewTodoSelfParentError(wrappedError error) *AppError {
	msg := "A todo cannot be its own parent."
	return &AppError{
		Type:         Validation,
		Title:        "Invalid Todo Parent",
		Message:      msg,
		wrappedError: wrappedError,
	}
}
