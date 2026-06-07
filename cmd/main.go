package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/shah-dhwanil/tasker/internal/app"
	"go.uber.org/zap"
)

const DefaultContextTimeout = 20
// @title           Tasker API
// @version         1.0
// @description     Task management API
// @host            localhost:8000
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
func main(){
	services,err:= app.NewServices()
	if err != nil {
		fmt.Println("failed to initialize app:", err)
		return
	}
	srv:= app.NewServer(services)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// Start server
	go func() {
		if err = srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			services.Observability().Logging().Logger().Fatal("Failed to Start the server",zap.Error(err))
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)

	if err = srv.Shutdown(ctx); err != nil {
		services.Observability().Logging().Logger().Fatal("Error while shutting down http server",zap.Error(err))
	}
	services.Shutdown()
	
	stop()
	cancel()
}