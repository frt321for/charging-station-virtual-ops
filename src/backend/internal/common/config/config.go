package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// Config contains runtime settings for the API service.
type Config struct {
	Environment       string
	HTTPHost          string
	HTTPPort          string
	PostgresHost      string
	PostgresPort      string
	PostgresDB        string
	PostgresUser      string
	PostgresPass      string
	PostgresSSLMode   string
	ValkeyAddr        string
	ValkeyPassword    string
	ValkeyDB          int
	NATSURL           string
	ModelScopeBaseURL string
	ModelScopeModel   string
	ModelScopeAPIKey  string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Environment:       getEnv("APP_ENV", "development"),
		HTTPHost:          getEnv("APP_HOST", "127.0.0.1"),
		HTTPPort:          getEnv("APP_PORT", "8080"),
		PostgresHost:      getEnv("POSTGRES_HOST", "127.0.0.1"),
		PostgresPort:      getEnv("POSTGRES_PORT", "15432"),
		PostgresDB:        getEnv("POSTGRES_DB", "charging_ops"),
		PostgresUser:      getEnv("POSTGRES_USER", "charging_ops"),
		PostgresPass:      getEnv("POSTGRES_PASSWORD", ""),
		PostgresSSLMode:   getEnv("POSTGRES_SSLMODE", "disable"),
		ValkeyAddr:        getEnv("VALKEY_ADDR", "127.0.0.1:16379"),
		ValkeyPassword:    getEnv("VALKEY_PASSWORD", ""),
		ValkeyDB:          getEnvInt("VALKEY_DB", 0),
		NATSURL:           getEnv("NATS_URL", "nats://127.0.0.1:14222"),
		ModelScopeBaseURL: getEnv("MODELSCOPE_BASE_URL", "https://api-inference.modelscope.cn/v1"),
		ModelScopeModel:   getEnv("MODELSCOPE_MODEL", "deepseek-ai/DeepSeek-V3.2"),
		ModelScopeAPIKey:  getEnv("MODELSCOPE_API_KEY", ""),
	}
}

// DatabaseURL returns a PostgreSQL connection string without logging it.
func (c Config) DatabaseURL() string {
	user := url.QueryEscape(c.PostgresUser)
	password := url.QueryEscape(c.PostgresPass)
	database := url.PathEscape(c.PostgresDB)
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		c.PostgresHost,
		c.PostgresPort,
		database,
		c.PostgresSSLMode,
	)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
