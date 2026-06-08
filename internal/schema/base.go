package schema

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/shah-dhwanil/tasker/internal/validation"
)

func init() {
    validation.RegisterCustomTypeFunc(
        func(field reflect.Value) any {
            n, ok := field.Interface().(Nullable[any])
            if !ok || !n.IsExplicitlySet {
                return nil
            }

            return n.Data
        },
        Nullable[any]{},
    )
}

type NullableField interface {
    IsSet() bool
    Value() any
}

type Nullable[T any] struct {
	Data T
	IsExplicitlySet bool
}

func (n Nullable[T]) IsSet() bool {
	return n.IsExplicitlySet
}

func (n Nullable[T]) Value() any {
	return n.Data
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.IsExplicitlySet = true
    if bytes.Equal(data, []byte("null")) {
        var zero T
        n.Data = zero
        return nil
    }
    return json.Unmarshal(data, &n.Data)
}

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