package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"particleapp/internal/auth"
	"particleapp/internal/config"
	"particleapp/internal/domain/particle"
	"particleapp/internal/domain/user"
	"particleapp/internal/filestorage"
	"particleapp/internal/httpapi"
	"particleapp/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	store, err := postgres.New(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("Ошибка подключения к PostgreSQL: ", err)
	}
	log.Println("Подключение к PostgreSQL установлено")

	files, err := filestorage.NewMinIO(cfg.MinIO)
	if err != nil {
		log.Fatal("Ошибка подключения к MinIO: ", err)
	}
	log.Printf("MinIO: бакет %s доступен", files.Bucket())

	authContext, cancelAuth := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelAuth()
	tokens, err := auth.New(authContext, cfg.Auth)
	if err != nil {
		log.Fatal("Ошибка подключения к Redis: ", err)
	}
	defer tokens.Close()
	log.Printf("Redis: %s доступен", cfg.Auth.RedisAddr)

	particleUsecase := particle.New(store, files)
	userUsecase := user.New(store, tokens)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Particles: particleUsecase,
		Users:     userUsecase,
		Store:     store,
		Files:     files,
		Auth:      tokens,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Веб-сервис API: http://localhost%s/api", cfg.HTTPAddr)
	log.Printf("Swagger UI: http://localhost%s/swagger/index.html", cfg.HTTPAddr)
	log.Fatal(server.ListenAndServe())
}
