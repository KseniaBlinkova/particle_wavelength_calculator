package main

import (
	"log"
	"net/http"
	"time"

	"particleapp/internal/app/particlehandler"
)

func main() {
	particleServer := &http.Server{
		Addr:              ":8081",
		Handler:           particlehandler.NewParticleMux(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Сервер запущен на http://localhost:8081")
	log.Fatal(particleServer.ListenAndServe())
}
