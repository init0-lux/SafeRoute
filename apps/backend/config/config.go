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
	AppName                        string
	Environment                    string
	Port                           string
	DatabaseURL                    string
	JWTAccessSecret                string
	JWTRefreshSecret               string
	JWTAccessTTL                   time.Duration
	JWTRefreshTTL                  time.Duration
	AuthAccessCookieName           string
	AuthRefreshCookieName          string
	AuthCookieDomain               string
	AuthCookieSameSite             string
	AuthCookieSecure               bool
	SOSViewerBaseURL               string
	EvidenceStorageRoot            string
	MaxEvidenceSizeBytes           int64
	ReportsNearbyDefaultLimit      int
	ReportsNearbyMaxLimit          int
	ReportsNearbyMaxRadiusM        float64
	SafetyDefaultRadiusM           float64
	SafetyMaxRadiusM               float64
	SafetyRecentWindow             time.Duration
	SafetyTimeRiskTimezone         string
	SafetyTimeRiskHighStartHour    int
	SafetyTimeRiskHighEndHour      int
	SafetyTimeRiskMorningStartHour int
	SafetyTimeRiskMorningEndHour   int
	SafetyTimeRiskEveningStartHour int
	SafetyTimeRiskEveningEndHour   int
	GoogleRoutesAPIKey             string
	GoogleRoutesBaseURL            string
	SafetyRouteCorridorRadiusM     float64
	SafetyRouteSegmentLengthM      float64
	SafetyRouteMaxDistanceM        float64
	SorobanRPCURL                  string
	SorobanContractID              string
	SorobanNetwork                 string
	SorobanSourceIdentity          string
	SorobanCLIBinary               string
	WorkerCleanupInterval          time.Duration
	WorkerCleanupMaxAge            time.Duration
	WorkerSafetyRefreshInterval    time.Duration
	WorkerIPFSPollInterval         time.Duration
	WorkerIndexerPollInterval      time.Duration
	WorkerIndexerLookbackLedgers   uint32
	WorkerRelayQueueSize           int
}

// Load returns runtime config using environment variables with local defaults.
func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	environment := getEnv("APP_ENV", "development")

	return Config{
		AppName:                        getEnv("APP_NAME", "SafeRoute Backend"),
		Environment:                    environment,
		Port:                           getEnv("PORT", "8080"),
		DatabaseURL:                    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/saferoute?sslmode=disable"),
		JWTAccessSecret:                getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret:               getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		JWTAccessTTL:                   getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:                  getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		AuthAccessCookieName:           getEnv("AUTH_ACCESS_COOKIE_NAME", "saferoute_access"),
		AuthRefreshCookieName:          getEnv("AUTH_REFRESH_COOKIE_NAME", "saferoute_refresh"),
		AuthCookieDomain:               getEnv("AUTH_COOKIE_DOMAIN", ""),
		AuthCookieSameSite:             getEnv("AUTH_COOKIE_SAME_SITE", "Lax"),
		AuthCookieSecure:               getBoolEnv("AUTH_COOKIE_SECURE", environment == "production"),
		SOSViewerBaseURL:               getEnv("SOS_VIEWER_BASE_URL", "http://localhost:8080/api/v1/sos/viewer/stream"),
		EvidenceStorageRoot:            getEnv("EVIDENCE_STORAGE_ROOT", "/tmp/saferoute-evidence"),
		MaxEvidenceSizeBytes:           getInt64Env("MAX_EVIDENCE_SIZE_BYTES", 10485760),
		ReportsNearbyDefaultLimit:      getIntEnv("REPORTS_NEARBY_DEFAULT_LIMIT", 20),
		ReportsNearbyMaxLimit:          getIntEnv("REPORTS_NEARBY_MAX_LIMIT", 50),
		ReportsNearbyMaxRadiusM:        getFloatEnv("REPORTS_NEARBY_MAX_RADIUS_METERS", 5000),
		SafetyDefaultRadiusM:           getFloatEnv("SAFETY_DEFAULT_RADIUS_METERS", 500),
		SafetyMaxRadiusM:               getFloatEnv("SAFETY_MAX_RADIUS_METERS", 3000),
		SafetyRecentWindow:             getDurationEnv("SAFETY_RECENT_WINDOW", 6*time.Hour),
		SafetyTimeRiskTimezone:         getEnv("SAFETY_TIME_RISK_TIMEZONE", "UTC"),
		SafetyTimeRiskHighStartHour:    getIntEnv("SAFETY_TIME_RISK_HIGH_START_HOUR", 22),
		SafetyTimeRiskHighEndHour:      getIntEnv("SAFETY_TIME_RISK_HIGH_END_HOUR", 5),
		SafetyTimeRiskMorningStartHour: getIntEnv("SAFETY_TIME_RISK_MORNING_START_HOUR", 5),
		SafetyTimeRiskMorningEndHour:   getIntEnv("SAFETY_TIME_RISK_MORNING_END_HOUR", 7),
		SafetyTimeRiskEveningStartHour: getIntEnv("SAFETY_TIME_RISK_EVENING_START_HOUR", 20),
		SafetyTimeRiskEveningEndHour:   getIntEnv("SAFETY_TIME_RISK_EVENING_END_HOUR", 22),
		GoogleRoutesAPIKey:             getEnv("GOOGLE_ROUTES_API_KEY", "hehe"),
		GoogleRoutesBaseURL:            getEnv("GOOGLE_ROUTES_BASE_URL", "https://routes.googleapis.com/directions/v2:computeRoutes"),
		SafetyRouteCorridorRadiusM:     getFloatEnv("SAFETY_ROUTE_CORRIDOR_RADIUS_METERS", 75),
		SafetyRouteSegmentLengthM:      getFloatEnv("SAFETY_ROUTE_SEGMENT_LENGTH_METERS", 150),
		SafetyRouteMaxDistanceM:        getFloatEnv("SAFETY_ROUTE_MAX_DISTANCE_METERS", 10000),
		SorobanRPCURL:                  getEnv("SOROBAN_RPC_URL", ""),
		SorobanContractID:              getEnv("SOROBAN_CONTRACT_ID", ""),
		SorobanNetwork:                 getEnv("SOROBAN_NETWORK", "testnet"),
		SorobanSourceIdentity:          getEnv("SOROBAN_SOURCE_IDENTITY", "backend-relayer"),
		SorobanCLIBinary:               getEnv("SOROBAN_CLI_BINARY", "stellar"),
		WorkerCleanupInterval:          getDurationEnv("WORKER_CLEANUP_INTERVAL", time.Hour),
		WorkerCleanupMaxAge:            getDurationEnv("WORKER_CLEANUP_MAX_AGE", 24*time.Hour),
		WorkerSafetyRefreshInterval:    getDurationEnv("WORKER_SAFETY_REFRESH_INTERVAL", 10*time.Minute),
		WorkerIPFSPollInterval:         getDurationEnv("WORKER_IPFS_POLL_INTERVAL", 15*time.Minute),
		WorkerIndexerPollInterval:      getDurationEnv("WORKER_INDEXER_POLL_INTERVAL", 10*time.Second),
		WorkerIndexerLookbackLedgers:   getUint32Env("WORKER_INDEXER_LOOKBACK_LEDGERS", 120),
		WorkerRelayQueueSize:           getIntEnv("WORKER_RELAY_QUEUE_SIZE", 100),
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

func getIntEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getInt64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getUint32Env(key string, fallback uint32) uint32 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return fallback
	}

	return uint32(parsed)
}

func getFloatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}
