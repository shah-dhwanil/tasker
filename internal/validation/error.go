package validation

import "github.com/go-playground/validator/v10"
import  pkgError "github.com/shah-dhwanil/tasker/internal/errors"

func TransformValidationError(err error)(bool,error){
	validationErr, ok := err.(validator.ValidationErrors)
	if !ok {
		return false, err
	}
	fieldErrors := make([]pkgError.FieldValidationError, len(validationErr))
	for _,err := range validationErr {
		fieldErrors = append(fieldErrors, *pkgError.NewFieldValidationError(err.Tag(), err.Error(), err.StructNamespace(), err.Value(), err.Param()))
	}
	return true, pkgError.NewValidationError("Validation failed for the provided input", fieldErrors, err)
}