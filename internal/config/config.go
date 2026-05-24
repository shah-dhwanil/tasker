package config

import (
	"fmt"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

var config *Config

// Init initializes the configuration by loading it from a file and environment variables.
func init() {
	var err error
	config, err = loadConfig()
	if err != nil {
		fmt.Printf("Error while loading config: %v\n", err)
		panic(fmt.Sprintf("Error while loading config: %v", err))
	}
}	
type Config struct {
	ServiceName string          `koanf:"service_name" validate:"required"`
	Environment string          `koanf:"environment" validate:"required"`
	NewRelic  NewRelicConfig `koanf:"new_relic" validate:"required"`
	Postgres	PostgresConfig  `koanf:"postgres" validate:"required"`
	Server      ServerConfig    `koanf:"server" validate:"required"`
}

func (payload *Config) Validate(validatorClient validation.ValidatorClient) error {
	err := validatorClient.Struct(payload)
	return err
}

// loadConfig loads the configuration from a YAML file and environment variables, validates it, and returns a Config struct.
func loadConfig() (*Config, error) {
	koanfClient := koanf.New(".")
	if err := koanfClient.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("%w", err)
	}
	if err := koanfClient.Load(env.Provider(".", env.Opt{
		Prefix: "TASKER_",
		TransformFunc: func(k, v string) (string, any) {
			return strings.ToLower(strings.TrimPrefix(k, "TASKER_")), v
		},
	}), nil); err != nil {
		return nil, fmt.Errorf("Error in loading Environment variables: %w", err)
	}
	config := &Config{}
	koanfClient.Unmarshal("", config)
	if err := validation.Validate(config); err != nil {
		return nil, fmt.Errorf("Validation Failed: %w\n", err)
	}
	return config, nil
}

// GetConfig returns the loaded configuration as a pointer to a Config struct.
func GetConfig() *Config {
	return config
}