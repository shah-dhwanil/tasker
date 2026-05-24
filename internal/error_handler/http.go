package errorhandler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/schema"
	pkgErrors "github.com/shah-dhwanil/tasker/internal/errors"
)

var mapErrorToStatusCode = map[pkgErrors.ErrorType]int{
	pkgErrors.Validation: 400,
	pkgErrors.Unknown:    500,
}

func GetStatusCodeForErrorType(errorType pkgErrors.ErrorType) int {
	if statusCode, exists := mapErrorToStatusCode[errorType]; exists {
		return statusCode
	}
	return 500 // Default to Internal Server Error if the error type is not mapped
}

func GetTypeForStatusCode(code int)pkgErrors.ErrorType{
	for errorType, statusCode := range mapErrorToStatusCode {
		if int(statusCode) == code {
			return errorType
		}
	}
	return pkgErrors.Unknown
}

func HandleError(err error) *schema.ErrorResponse {
	var appError *pkgErrors.AppError
	var validationError *pkgErrors.ValidationError
	var httpError *echo.HTTPError
	var databaseError *pkgErrors.DatabaseError
	
	switch {
	case errors.As(err, &appError):
		return &schema.ErrorResponse{
			StatusCode: GetStatusCodeForErrorType(appError.Type),
			Type: string(appError.Type),
			Title: appError.Title,
			Detail:    appError.Message,
			Resource: appError.Resource,
		}
	case errors.As(err, &validationError):
		errors := make([]any, len(validationError.FieldErrors))
		for i, v := range validationError.FieldErrors {
				errors[i] = v
		}
		return &schema.ErrorResponse{
			StatusCode: GetStatusCodeForErrorType(validationError.Type),
			Type: string(validationError.Type),
			Title: validationError.Title,
			Detail:    validationError.Message,
			Resource: validationError.Resource,
			Errors: errors,
		}
	case errors.As(err, &httpError):
		return &schema.ErrorResponse{
			StatusCode: httpError.Code,
			Type: string(GetTypeForStatusCode(httpError.Code)),
			Title: http.StatusText(httpError.Code),
			Detail:    httpError.Error(),
		}
	case errors.As(err, &databaseError):
		return &schema.ErrorResponse{
			StatusCode: 500,
			Type: string(pkgErrors.Unknown),
			Title: "Unkown Error",
			Detail:    "An unkown error occurred while processing the request.",
		}
	default:
		return &schema.ErrorResponse{
			StatusCode: 500,
			Type: string(pkgErrors.Unknown),
			Title: "Unkown Error",
			Detail:    "An unkown error occurred while processing the request.",
		}
	}
}