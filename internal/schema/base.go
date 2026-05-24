package schema

type Response[T any] struct {
	StatusCode uint16 `json:"statusCode"`
	Data       T      `json:"data"`
}

type PaginatedResponse[T any] struct {
	StatusCode uint16 `json:"statusCode"`
	Data       []T `json:"data"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}