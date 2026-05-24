package config

type NewRelicConfig struct {
	LicenseKey string `koanf:"license_key" validate:"required"`
	AppLogForwardingEnabled   bool   `koanf:"app_log_forwarding_enabled" validate:"required"`
	DistributedTracingEnabled bool   `koanf:"distributed_tracing_enabled" validate:"required"`
	DebugLogging              bool   `koanf:"debug_logging"`
}
