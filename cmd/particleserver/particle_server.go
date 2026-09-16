package main

import (
	"log"
	"net/http"
	"time"

	"particleapp/internal/app/particlehandler"
	"particleapp/internal/particlerepository"
)

func main() {
	// Подключение к БД
	if err := particlerepository.InitDB(); err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	log.Println(" Подключение к PostgreSQL установлено")

	particleServer := &http.Server{
		Addr:              ":8081",
		Handler:           particlehandler.NewParticleMux(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Сервер запущен на http://localhost:8081")
	log.Fatal(particleServer.ListenAndServe())
}
