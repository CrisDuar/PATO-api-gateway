package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Valkey   ValkeyConfig
	Services ServicesConfig
}

type ServerConfig struct {
	Port        string
	Environment string
	Name        string
}

type ValkeyConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type ServicesConfig struct {
	UserServiceURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("APP_PORT", "8081"),
			Environment: getEnv("APP_ENV", "development"),
			Name:        getEnv("APP_NAME", "PATO API Gateway"),
		},

		Valkey: ValkeyConfig{
			Addr:     getEnv("APP_VALKEY_ADDR", "localhost:6379"),
			Username: getEnv("APP_VALKEY_USER", ""),
			Password: getEnv("APP_VALKEY_PASSWORD", ""),
			DB:       getEnvInt("APP_VALKEY_DB", 0),
		},

		Services: ServicesConfig{
			UserServiceURL: getEnv("APP_USER_SERVICE_URL", "http://localhost:8080"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}

	return defaultValue
}
