package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName               string
	Environment           string
	Port                  string
	DatabaseURL           string
	JWTAccessSecret       string
	JWTRefreshSecret      string
	JWTAccessTTL          time.Duration
	JWTRefreshTTL         time.Duration
	AuthAccessCookieName  string
	AuthRefreshCookieName string
	AuthCookieDomain      string
	AuthCookieSameSite    string
	AuthCookieSecure      bool
}

// Load returns runtime config using environment variables with local defaults.
func Load() Config {
	// Attempt to load .env file from the current directory.
	// We ignore error as .env might not exist in some environments (e.g. production)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	environment := getEnv("APP_ENV", "development")

	return Config{
		AppName:               getEnv("APP_NAME", "SafeRoute Backend"),
		Environment:           environment,
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/saferoute?sslmode=disable"),
		JWTAccessSecret:       getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret:      getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		JWTAccessTTL:          getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:         getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		AuthAccessCookieName:  getEnv("AUTH_ACCESS_COOKIE_NAME", "saferoute_access"),
		AuthRefreshCookieName: getEnv("AUTH_REFRESH_COOKIE_NAME", "saferoute_refresh"),
		AuthCookieDomain:      getEnv("AUTH_COOKIE_DOMAIN", ""),
		AuthCookieSameSite:    getEnv("AUTH_COOKIE_SAME_SITE", "Lax"),
		AuthCookieSecure:      getBoolEnv("AUTH_COOKIE_SECURE", environment == "production"),
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

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
