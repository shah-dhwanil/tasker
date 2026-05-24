package observability

import "github.com/shah-dhwanil/tasker/internal/config"

type ObservabilityService struct {
	loggingService *LoggingService
	newRelicService *NewRelicService
}

func New(config *config.Config) (*ObservabilityService,error) {
	loggerService,err:= newLoggingService(config)
	if err != nil {
		return nil, err
	}
	newRelicService, err := newNewRelicService(config, loggerService)
	if err != nil {
		return nil, err
	}
	return &ObservabilityService{
		loggingService: loggerService,
		newRelicService: newRelicService,
	}, nil
}

func GetDefaultObservabilityService() *ObservabilityService {
	loggerService := &LoggingService{logger: getDefaultLogger()}
	newRelicService := &NewRelicService{nrApp: nil}
	return &ObservabilityService{
		loggingService: loggerService,
		newRelicService: newRelicService,
	}
}

func (o *ObservabilityService) Logging() *LoggingService {
	return o.loggingService
}

func (o *ObservabilityService) NewRelic() *NewRelicService {
	return o.newRelicService
}

func (o *ObservabilityService) Shutdown() {
	o.NewRelic().Shutdown()
}