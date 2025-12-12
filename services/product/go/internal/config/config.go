package config

import (
	"os"
	"strconv"
)

type Config struct {
	// App
	AppPort string

	// Database
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// Redis
	RedisHost string
	RedisPort string

	// Cache
	EnableCache bool
	CacheTTL    int

	// gRPC
	GRPCEnabled bool
	GRPCPort    string

	// OpenTelemetry
	OTelEnabled          bool
	OTelServiceName      string
	OTelExporterEndpoint string
}

func Load() *Config {
	return &Config{
		// App
		AppPort: getEnv("APP_PORT", "8002"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "flash_deals"),
		DBUser:     getEnv("DB_USER", "flash"),
		DBPassword: getEnv("DB_PASSWORD", "flash1234"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// Cache
		EnableCache: getEnvBool("ENABLE_CACHE", true),
		CacheTTL:    getEnvInt("CACHE_TTL", 60),

		// gRPC
		GRPCEnabled: getEnvBool("GRPC_ENABLED", false),
		GRPCPort:    getEnv("GRPC_PORT", "50051"),

		// OpenTelemetry
		OTelEnabled:          getEnvBool("OTEL_ENABLED", false),
		OTelServiceName:      getEnv("OTEL_SERVICE_NAME", "product-service"),
		OTelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
	}
}

func (c *Config) DatabaseURL() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
