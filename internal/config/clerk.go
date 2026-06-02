package config

type ClerkConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required"`
}