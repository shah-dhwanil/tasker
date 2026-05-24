package observability

import (
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shah-dhwanil/tasker/internal/config"
	"go.uber.org/zap"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzap"
)

type NewRelicService struct{
	nrApp *newrelic.Application
}

func newNewRelicService(config *config.Config, loggerService *LoggingService) (*NewRelicService, error) {
	var configOptions []newrelic.ConfigOption

	configOptions = append(configOptions,
		newrelic.ConfigAppName(config.ServiceName),
		newrelic.ConfigLicense(config.NewRelic.LicenseKey),
		newrelic.ConfigAppLogForwardingEnabled(config.NewRelic.AppLogForwardingEnabled),
		newrelic.ConfigDistributedTracerEnabled(config.NewRelic.DistributedTracingEnabled),
	)
	// Add debug logging only if explicitly enabled
	if config.NewRelic.DebugLogging {
		configOptions = append(configOptions, newrelic.ConfigDebugLogger(os.Stdout))
	}
	nrApp, err := newrelic.NewApplication(configOptions...)
	if err != nil {
		return nil, fmt.Errorf("Error while setting up New Relic:- %w", err)
	}
	backgroundCore, err := nrzap.WrapBackgroundCore(loggerService.Logger().Core(), nrApp)
	if err != nil && err != nrzap.ErrNilApp {
	    panic(err)
	}
	
	backgroundLogger := zap.New(backgroundCore)
	loggerService.logger = backgroundLogger

	return &NewRelicService{nrApp: nrApp}, nil
}

func (n *NewRelicService) Application() *newrelic.Application {
	return n.nrApp
}

func (n *NewRelicService) Shutdown() {
	n.nrApp.Shutdown(10 * 1000) // wait for 10 seconds to allow any pending data to be sent to New Relic before shutting down
}