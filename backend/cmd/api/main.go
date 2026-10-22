package main

import (
	"log"
	"net/http"

	"tripplanner/backend/internal/config"
	"tripplanner/backend/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("API listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, httpapi.NewRouter(httpapi.Dependencies{})); err != nil {
		log.Fatal(err)
	}
}
