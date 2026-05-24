package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

func RequestID() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return uuid.NewString()
		},
		TargetHeader: RequestIDHeader,
		RequestIDHandler: func(ctx echo.Context, s string) {
			ctx.Response().Header().Add(RequestIDHeader,s)
			ctx.Set(RequestIDKey, s)
		},
	})
}