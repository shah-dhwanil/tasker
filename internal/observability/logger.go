package observability

import (
	"context"
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shah-dhwanil/tasker/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzap"
)

// LoggerKey is the key used to store the logger in the context.
const LoggerKey = "zap_logger"

// Logger is a wrapper around zap.Logger to provide additional functionality if needed in the future.
type Logger  = *zap.Logger

// LoggingService is a service that provides a logger instance configured based on the environment.
type LoggingService struct {
	logger Logger
}

func newLoggingService(config *config.Config) (*LoggingService,error) {
	env := config.Environment
	var isDevelopment bool
	if env == "production" {
		isDevelopment = false
	} else {
		isDevelopment = true
	}
	var encoderCfg zapcore.EncoderConfig
	var disableCaller bool
	var disableStackTrace bool
	var logLevel zapcore.Level
	var encoding string
	if isDevelopment {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		disableCaller = false
		disableStackTrace = false
		logLevel = zap.InfoLevel
		encoding = "console"
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
		disableCaller = true
		disableStackTrace = true
		logLevel = zap.InfoLevel
		encoding = "json"
	}
	encoderCfg.TimeKey = "timestamp"
    encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    zapconfig := zap.Config{
        Level:             zap.NewAtomicLevelAt(logLevel),
        Development:       isDevelopment,
        DisableCaller:     disableCaller,
        DisableStacktrace: disableStackTrace,
        Sampling:          nil,
        Encoding:          encoding,
        EncoderConfig:     encoderCfg,
        OutputPaths: []string{
            "stderr",
        },
        ErrorOutputPaths: []string{
            "stderr",
        },
        InitialFields: map[string]any{
           	"service":config.ServiceName,
           	"environment":config.Environment,
            "pid": os.Getpid(),
        },
    }
    logger,err := zapconfig.Build()
    if err != nil {
    	return nil, fmt.Errorf("Error while building logger: %v\n", err)
	}
    return &LoggingService{
		logger: logger,
	},nil
}

func getDefaultLogger() Logger {
	return zap.NewNop()
}

func (ls *LoggingService) Logger() Logger {
	return ls.logger
}

func (ls *LoggingService) SugarLogger() *zap.SugaredLogger {
	return ls.logger.Sugar()
}

func (ls *LoggingService) Sync() {
	err := ls.logger.Sync()
	if err != nil {
		ls.logger.Error("Error while syncing logger", zap.Error(err))
	}
}


func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(LoggerKey).(Logger); ok {
		return logger
	}
	// Fallback to a basic logger if not found
	return zap.NewNop()
}

func AttachtoContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

func WithContext(ctx context.Context,logger Logger) Logger {
	// TODO: Add new relic transaction fields to the logger
	core:= logger.Core()
	txnCore, err := nrzap.WrapTransactionCore(core, newrelic.FromContext(ctx))
	if err != nil && err != nrzap.ErrNilTxn {
		logger.Error("Error while wrapping logger with new relic transaction", zap.Error(err))
		return logger
	}
	return zap.New(txnCore)
}