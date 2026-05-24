package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/shah-dhwanil/tasker/internal/observability"
)

func NewRelic(app *observability.ObservabilityService) echo.MiddlewareFunc {
	return nrecho.Middleware(app.NewRelic().Application())
}