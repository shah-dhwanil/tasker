package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"go.uber.org/zap"
)

func RequestLogger(observabilityService *observability.ObservabilityService) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogError:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger := observability.FromContext(c.Request().Context())
			if v.Error == nil {
				logger.Info("Request fullfiled Succesfully", zap.Int("status", v.Status), zap.Duration("latency", v.Latency))
			} else {
				logger.Error("Request failed with error", zap.Int("status", v.Status), zap.Duration("latency", v.Latency), zap.Error(v.Error))
			}
			return nil
		},
	})
}

func AttachContextLogger(logService *observability.ObservabilityService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := observability.WithContext(c.Request().Context(),logService.Logging().Logger())
			nLogger := logger.With(
				zap.String("request_id", c.Get(RequestIDKey).(string)),
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.String("ip", c.RealIP()),
			)
			ctx := observability.AttachtoContext(c.Request().Context(), nLogger)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}