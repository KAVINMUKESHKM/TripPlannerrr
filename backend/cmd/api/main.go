package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tripplanner/backend/internal/auth"
	"tripplanner/backend/internal/config"
	"tripplanner/backend/internal/httpapi"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatal("could not load backend .env")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("configure database connection")
	}
	defer pool.Close()
	authHandler := auth.NewHandler(pool, auth.Options{CookieSecure: cfg.CookieSecure, AllowedOrigin: cfg.FrontendURL})
	log.Printf("API listening on :%s", cfg.Port)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.NewRouter(httpapi.Dependencies{Ready: pool.Ping, Auth: authHandler.Routes()}), ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
