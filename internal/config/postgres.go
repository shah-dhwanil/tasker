package config

type PostgresConfig struct {
	DSN             string `koanf:"dsn" validate:"required"`
	PingTimeout     uint    `koanf:"ping_timeout" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MinIdleConns    int    `koanf:"min_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}