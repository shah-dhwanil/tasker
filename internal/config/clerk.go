package config

type ClerkConfig struct {
	SECRET_KEY string `koanf:"secret_key" validate:"required"`
}