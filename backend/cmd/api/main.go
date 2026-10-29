package main

import (
	"context"
	"log"
	"net/http"
	"strings"
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
		log.Fatalf("could not load backend .env: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid backend configuration: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("could not configure database connection (check DATABASE_URL): %v", err)
	}
	defer pool.Close()
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := pool.Ping(dbCtx); err != nil {
		dbCancel()
		log.Fatalf("database connection failed (check DATABASE_URL and run migrations): %v", err)
	}
	dbCancel()
	log.Printf("database connection ready")
	go cleanupExpiredSecurityRecords(pool)
	var googleOAuth *oauth2.Config
	var googleVerifier auth.IdentityVerifier
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		redirectURL := cfg.GoogleRedirectURL
		if redirectURL == "" {
			redirectURL = strings.TrimRight(cfg.FrontendURL, "/") + "/api/v1/auth/google/callback"
		}
		log.Printf("Google OAuth callback URI: %s", redirectURL)
		googleOAuth = &oauth2.Config{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret, RedirectURL: redirectURL, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}, Scopes: []string{"openid", "email", "profile"}}
		googleVerifier = auth.NewGoogleOIDCVerifier()
	}
	authHandler := auth.NewHandler(pool, auth.Options{CookieSecure: cfg.CookieSecure, AllowedOrigin: cfg.FrontendURL, GoogleOAuth: googleOAuth, GoogleVerifier: googleVerifier, FrontendURL: cfg.FrontendURL, SMTP: auth.SMTPConfig{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}})
	placeRepository := places.NewRepository(pool)
	placeHandler := places.NewHandler(placeRepository, providers.Geocoder{APIKey: cfg.OpenCageAPIKey}, providers.Weather{APIKey: cfg.OpenWeatherAPIKey}, providers.Router{BaseURL: cfg.OSRMBaseURL}, providers.Restaurants{APIKey: cfg.SerpAPIKey})
	tripHandler := trips.NewHandler(trips.NewRepository(pool), authHandler)
	rateLimiter, err := httpapi.NewPostgresRateLimiter(pool, cfg.TrustedProxies)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("API starting: address=:%s environment=%s frontend=%s", cfg.Port, valueOr(cfg.Environment, "development"), valueOr(cfg.FrontendURL, "not configured"))
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(httpapi.Dependencies{Ready: pool.Ping, Limiter: rateLimiter, Auth: authHandler.Routes(), Places: placeHandler.Routes(), Trips: tripHandler.Routes()}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func cleanupExpiredSecurityRecords(pool *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at <= now()")
		_, _ = pool.Exec(ctx, "DELETE FROM oauth_states WHERE expires_at <= now()")
		_, _ = pool.Exec(ctx, "DELETE FROM password_reset_tokens WHERE expires_at <= now()")
		_, _ = pool.Exec(ctx, "DELETE FROM api_rate_limits WHERE window_started <= now() - interval '5 minutes'")
		cancel()
	}
}
