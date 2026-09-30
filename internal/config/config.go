package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr    string
	DatabaseDSN string
	MinIO       MinIOConfig
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

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
