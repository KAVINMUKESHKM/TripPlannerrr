package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"tripplanner/backend/internal/auth"
	"tripplanner/backend/internal/config"
	"tripplanner/backend/internal/httpapi"
	"tripplanner/backend/internal/places"
	"tripplanner/backend/internal/providers"
	"tripplanner/backend/internal/trips"
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
	var googleOAuth *oauth2.Config
	var googleVerifier auth.IdentityVerifier
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		redirectURL := cfg.GoogleRedirectURL
		if redirectURL == "" {
			redirectURL = fmt.Sprintf("http://localhost:%s/api/v1/auth/google/callback", cfg.Port)
		}
		googleOAuth = &oauth2.Config{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret, RedirectURL: redirectURL, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}, Scopes: []string{"openid", "email", "profile"}}
		googleVerifier = auth.NewGoogleOIDCVerifier()
	}
	authHandler := auth.NewHandler(pool, auth.Options{CookieSecure: cfg.CookieSecure, AllowedOrigin: cfg.FrontendURL, GoogleOAuth: googleOAuth, GoogleVerifier: googleVerifier, FrontendURL: cfg.FrontendURL})
	placeRepository := places.NewRepository(pool)
	placeHandler := places.NewHandler(placeRepository, providers.Geocoder{APIKey: cfg.OpenCageAPIKey}, providers.Weather{APIKey: cfg.OpenWeatherAPIKey}, providers.Router{BaseURL: cfg.OSRMBaseURL})
	tripHandler := trips.NewHandler(trips.NewRepository(pool), authHandler)
	log.Printf("API listening on :%s", cfg.Port)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.NewRouter(httpapi.Dependencies{Ready: pool.Ping, Auth: authHandler.Routes(), Places: placeHandler.Routes(), Trips: tripHandler.Routes()}), ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
