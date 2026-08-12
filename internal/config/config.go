package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server      ServerConfig
	UserService ServiceConfig
	Valkey      ValkeyConfig
	CORS        CORSConfig
}

type ServerConfig struct {
	Port        string
	Environment string
	Name        string
}

type ServiceConfig struct {
	BaseURL string
}

type ValkeyConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type CORSConfig struct {
	AllowedOrigin string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("APP_PORT", "8080"),
			Environment: getEnv("APP_ENV", "development"),
			Name:        getEnv("APP_NAME", "PATO API Gateway"),
		},

		UserService: ServiceConfig{
			BaseURL: getEnv(
				"USER_SERVICE_URL",
				"http://localhost:8081",
			),
		},

		Valkey: ValkeyConfig{
			Addr:     getEnv("APP_VALKEY_ADDR", "localhost:6379"),
			Username: getEnv("APP_VALKEY_USER", ""),
			Password: getEnv("APP_VALKEY_PASSWORD", ""),
			DB:       getEnvInt("APP_VALKEY_DB", 0),
		},

		CORS: CORSConfig{
			AllowedOrigin: getEnv(
				"APP_CORS_ORIGIN",
				"http://localhost:3000",
			),
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
