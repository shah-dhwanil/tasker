package schema

type ErrorResponse struct {
	StatusCode int `json:"statusCode" format:"int32" example:"400" doc:"HTTP status code"`
	
	Type string `json:"type,omitempty" format:"uri" default:"about:blank" example:"https://example.com/errors/example" doc:"A URI reference to human-readable documentation for the error."`
	// Title provides a short static summary of the problem. Huma will default this
	// to the HTTP response status code text if not present.
	Title string `json:"title,omitempty" example:"Bad Request" doc:"A short, human-readable summary of the problem type. This value should not change between occurrences of the error."`

	// Detail is an explanation specific to this error occurrence.
	Detail string `json:"detail,omitempty" example:"Property foo is required but is missing." doc:"A human-readable explanation specific to this occurrence of the problem."`

	Errors []any `json:"errors,omitempty" doc:"An optional array of error details, which can provide additional information about specific errors that occurred."`

	Resource *string `json:"resource,omitempty" example:"User" doc:"The resource associated with the error, if applicable."`

}