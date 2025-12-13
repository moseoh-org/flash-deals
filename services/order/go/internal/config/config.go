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

	// Async Order Processing
	AsyncOrderEnabled bool

	// Product Service gRPC
	ProductGRPCHost string
	ProductGRPCPort string

	// OpenTelemetry
	OTelEnabled          bool
	OTelServiceName      string
	OTelExporterEndpoint string
}

func Load() *Config {
	return &Config{
		// App
		AppPort: getEnv("APP_PORT", "8003"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "flash_deals"),
		DBUser:     getEnv("DB_USER", "flash"),
		DBPassword: getEnv("DB_PASSWORD", "flash1234"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "redis"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// Async Order Processing
		AsyncOrderEnabled: getEnvBool("ASYNC_ORDER_ENABLED", false),

		// Product Service gRPC
		ProductGRPCHost: getEnv("PRODUCT_GRPC_HOST", "product"),
		ProductGRPCPort: getEnv("PRODUCT_GRPC_PORT", "50051"),

		// OpenTelemetry
		OTelEnabled:          getEnvBool("OTEL_ENABLED", false),
		OTelServiceName:      getEnv("OTEL_SERVICE_NAME", "order-service"),
		OTelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
	}
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

func (c *Config) DatabaseURL() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

func (c *Config) ProductGRPCAddr() string {
	return c.ProductGRPCHost + ":" + c.ProductGRPCPort
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
