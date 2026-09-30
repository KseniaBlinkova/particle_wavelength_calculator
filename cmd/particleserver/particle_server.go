package main

import (
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

	particleUsecase := particle.New(store, files)
	userUsecase := user.New(store)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Particles: particleUsecase,
		Users:     userUsecase,
		Store:     store,
		Files:     files,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	currentUserID := auth.Get().ID
	log.Printf("Текущий пользователь (константа): #%d", currentUserID)

	log.Printf("Веб-сервис API: http://localhost%s/api", cfg.HTTPAddr)
	log.Fatal(server.ListenAndServe())
}
