package app

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	errorhandler "github.com/shah-dhwanil/tasker/internal/error_handler"
	"github.com/shah-dhwanil/tasker/internal/handler"
	"github.com/shah-dhwanil/tasker/internal/middleware"
	"github.com/shah-dhwanil/tasker/internal/repository"
	"github.com/shah-dhwanil/tasker/internal/routes"
	"github.com/shah-dhwanil/tasker/internal/service"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *echo.Echo
	services  *Services
}

func NewServer(services *Services) *Server {
	server := echo.New()
	server.Server.IdleTimeout = time.Duration(services.Config().Server.IdleTimeout) * time.Second
	server.Server.WriteTimeout = time.Duration(services.Config().Server.WriteTimeout) * time.Second
	server.Server.ReadTimeout = time.Duration(services.Config().Server.ReadTimeout) * time.Second
	server.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		res:= errorhandler.HandleError(err)
		c.JSON(res.StatusCode,res)
	}
	middleware.Setup(server, services.Config(),services.Observability())
	handlers := handler.New(service.New(repository.New()))
	routes.RegisterRoutes(server,handlers)
	return &Server{
		httpServer: server,
		services:  services,
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return fmt.Errorf("Server not initialized")
	}
	fmt.Printf("Starting server on port %s in %s environment\n", s.services.Config().Server.Port, s.services.Config().Environment)
	s.services.Observability().Logging().Logger().
		Debug("Starting Server",
			zap.String("port", s.services.Config().Server.Port),
			zap.String("environment", s.services.Config().Environment),
		)
	return s.httpServer.Start(":" + s.services.Config().Server.Port)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	return nil
}