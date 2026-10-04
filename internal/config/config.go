package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"particleapp/internal/auth"
)

type Config struct {
	HTTPAddr    string
	DatabaseDSN string
	MinIO       MinIOConfig
	Auth        auth.Config
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	PublicURL       string
}

func Load() Config {
	// docker-compose.yml публикует MinIO API на порту 9010.
	minioEndpoint := env("MINIO_ENDPOINT", "localhost:9010")
	minioBucket := env("MINIO_BUCKET", "de-broglie")
	minioUseSSL := env("MINIO_USE_SSL", "false") == "true"
	minioScheme := "http"
	if minioUseSSL {
		minioScheme = "https"
	}
	minioPublicURL := env("MINIO_PUBLIC_URL", fmt.Sprintf("%s://%s/%s", minioScheme, minioEndpoint, minioBucket))

	return Config{
		HTTPAddr:    env("HTTP_ADDR", ":8081"),
		DatabaseDSN: env("DATABASE_DSN", "host=localhost user=particle_user password=particle_pass dbname=particle_db port=5433 sslmode=disable"),
		Auth: auth.Config{
			Secret:        env("JWT_SECRET", "lab4-development-secret-change-me"),
			TTL:           envDuration("JWT_TTL", time.Hour),
			RedisAddr:     env("REDIS_ADDR", "localhost:6380"),
			RedisPassword: env("REDIS_PASSWORD", ""),
			RedisDB:       envInt("REDIS_DB", 0),
		},
		MinIO: MinIOConfig{
			Endpoint:        minioEndpoint,
			AccessKeyID:     env("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: env("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:          minioBucket,
			UseSSL:          minioUseSSL,
			PublicURL:       minioPublicURL,
		},
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
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

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
