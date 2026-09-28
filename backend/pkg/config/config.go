// Package config loads application configuration from YAML and the environment.
package config

import (
	"mch_api/internal/app"
	"os"
	"strings"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type (
	// Config defines Config values.
	Config struct {
		ConnectionString string `mapstructure:"connection_string"`
		Port             string `mapstructure:"port"`
		CORSOrigins      string `mapstructure:"cors_origins"`
	}
)

// New loads an independently owned application configuration.
func New() *Config {
	loader := config.New("application", config.ParseEnv)
	loader.AddDriver(yaml.Driver)
	if err := loader.LoadFiles("config/dev.yaml"); err != nil {
		panic(app.Wrap(err, "configuration"))
	}
	var cfg Config
	if err := loader.Decode(&cfg); err != nil {
		panic(app.Wrap(err, "configuration"))
	}

	cfg.applyDefaults()
	cfg.applyEnv()
	return &cfg
}

func (c *Config) applyDefaults() {
	if c.ConnectionString == "" {
		c.ConnectionString = "postgresql://localhost:5432/postgres"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.CORSOrigins == "" {
		c.CORSOrigins = "http://localhost:8000"
	}
}

func (c *Config) applyEnv() {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		c.ConnectionString = value
	}
	if value := os.Getenv("PORT"); value != "" {
		c.Port = value
	}
	if value := os.Getenv("CORS_ORIGINS"); value != "" {
		c.CORSOrigins = value
	}
}

// Addr executes Addr behavior.
func (c Config) Addr() string {
	if strings.HasPrefix(c.Port, ":") {
		return c.Port
	}
	return ":" + c.Port
}

// AllowedOrigins executes AllowedOrigins behavior.
func (c Config) AllowedOrigins() []string {
	var origins []string
	for _, origin := range strings.Split(c.CORSOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
