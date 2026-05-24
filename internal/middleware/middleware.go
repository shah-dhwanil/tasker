package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/shah-dhwanil/tasker/internal/config"
	"github.com/shah-dhwanil/tasker/internal/observability"
)

func Setup(server *echo.Echo, config *config.Config, observabilityService *observability.ObservabilityService) {
	server.Use(CORS(config))
	server.Use(middleware.Secure())
	server.Use(RequestID())
	server.Use(NewRelic(observabilityService))
	server.Use(AttachContextLogger(observabilityService))
	server.Use(RateLimiter())
	server.Use(RequestLogger(observabilityService))
	server.Use(middleware.Recover())
}