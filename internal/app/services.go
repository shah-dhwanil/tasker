package app

import (
	"context"
	"fmt"

	"github.com/shah-dhwanil/tasker/internal/config"
	"github.com/shah-dhwanil/tasker/internal/database"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"go.uber.org/zap"
)

type Services struct {
	db database.PgPool
	observability *observability.ObservabilityService
	config *config.Config
}

func NewServices() (*Services,error) {
	config := config.GetConfig()
	observabilityService,err := observability.New(config)
	if err != nil {
		observabilityService = observability.GetDefaultObservabilityService() 
	}
	pool, err := database.NewPgPool(config)
	if err != nil {
		observabilityService.Logging().Logger().Fatal("Error while creating postgres connection pool",zap.Error(err))
		return nil, fmt.Errorf("Error while connecting to database: %w", err)
	}

	if err := database.Migrate(context.Background(), config, observabilityService.Logging().Logger()); err != nil {
		observabilityService.Logging().Logger().Fatal("Failed to apply migrations on postgres database",zap.Error(err))
		return nil, fmt.Errorf("Failed to migrate database: %w", err)
	}
	return &Services{
		db: pool,
		observability: observabilityService,
		config: config,
	}, nil
}

func (s *Services) Shutdown() {
	s.observability.Shutdown()
	s.db.Close()
}

func (s *Services) DB() database.PgPool {
	return s.db
}

func (s *Services) Observability() *observability.ObservabilityService {
	return s.observability
}

func (s *Services) Config() *config.Config{
	return s.config	
}