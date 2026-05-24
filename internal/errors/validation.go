package errors

import "fmt"

type ValidationError struct {
	AppError
	FieldErrors []FieldValidationError `json:"fieldErrors,omitempty"`
}

func (err *ValidationError) Unwrap() error {
	return err.wrappedError
}

func (err *ValidationError) Error() string {
	msg:= ""
	for _, fieldErr := range err.FieldErrors {
		msg += fmt.Sprintf("%v\n", fieldErr.Error())
	}
	return msg
}

type FieldValidationError struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	Value       any     `json:"value"`
	ConstraintValue  any `json:"constraintValue,omitempty"`
}
func (err *FieldValidationError) Error() string {
	val := fmt.Sprintf("Location:- %v| Type:- %v | Provided Value:-  %v", err.Location, err.Type, err.Value)
	if err.ConstraintValue != nil {
		val = fmt.Sprintf("%v | Bound Value:- %v", val, err.ConstraintValue)
	}
	return val
}

func NewFieldValidationError(err_type string, description string, location string, value any, constraintValue any) *FieldValidationError {
	return &FieldValidationError{
		Type: err_type,
		Description: description,
		Location: location,
		Value: value,
		ConstraintValue: constraintValue,
	}
}

func NewValidationError(message string, fieldErrors []FieldValidationError, orignalError error) *ValidationError{
	return &ValidationError{
		AppError: AppError{
			Type: Validation,
			Title: "Validation Error",
			Message: message,
			wrappedError: orignalError,
		},
		FieldErrors: fieldErrors,
	}	
}

func NewBindingError(orignalError error) *ValidationError{
	return &ValidationError{
		AppError: AppError{
			Type: Validation,
			Title: "Invalid Request",
			Message: "Failed to bind the request. Please ensure the request is well-formed and adheres to the expected schema.",
			wrappedError: orignalError,
		},
		FieldErrors: make([]FieldValidationError, 0),
	}	
}