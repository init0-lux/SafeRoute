package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppName     string
	Environment string
	Port        string
}

// Load returns runtime config using environment variables with local defaults.
func Load() Config {
	return Config{
		AppName:     getEnv("APP_NAME", "SafeRoute Backend"),
		Environment: getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
	}
}

func (c Config) Address() string {
	port := strings.TrimSpace(c.Port)
	if port == "" {
		port = "8080"
	}

	if strings.HasPrefix(port, ":") {
		return port
	}

	return fmt.Sprintf(":%s", port)
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
