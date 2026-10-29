package config

import (
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL         string
	MigratorDatabaseURL string
	ImporterDatabaseURL string
	OpenWeatherAPIKey   string
	SerpAPIKey          string
	OpenCageAPIKey      string
	OSRMBaseURL         string
	CookieSecure        bool
	Environment         string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	FrontendURL         string
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	SMTPFrom            string
	TrustedProxies      string
	Port                string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"), MigratorDatabaseURL: os.Getenv("MIGRATOR_DATABASE_URL"), ImporterDatabaseURL: os.Getenv("IMPORTER_DATABASE_URL"), OpenWeatherAPIKey: firstSet("OPENWEATHER_API_KEY", "OPEN_WEATHER_KEY"), SerpAPIKey: firstSet("SERP_API_KEY", "SERP_KEY"),
		OpenCageAPIKey: os.Getenv("OPENCAGE_API_KEY"), OSRMBaseURL: os.Getenv("OSRM_BASE_URL"),
		CookieSecure:   os.Getenv("COOKIE_SECURE") == "true",
		Environment:    os.Getenv("APP_ENV"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL: os.Getenv("GOOGLE_REDIRECT_URL"),
		FrontendURL:       os.Getenv("FRONTEND_URL"), Port: os.Getenv("PORT"),
		SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: os.Getenv("SMTP_PORT"), SMTPUsername: os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM"),
		TrustedProxies: os.Getenv("TRUSTED_PROXIES"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.Environment == "production" && !cfg.CookieSecure {
		return Config{}, errors.New("COOKIE_SECURE=true is required when APP_ENV=production")
	}
	if cfg.Environment == "production" && !strings.HasPrefix(cfg.FrontendURL, "https://") {
		return Config{}, errors.New("FRONTEND_URL must use HTTPS when APP_ENV=production")
	}
	if cfg.Environment == "production" && cfg.GoogleClientID != "" && !strings.HasPrefix(cfg.GoogleRedirectURL, "https://") {
		return Config{}, errors.New("GOOGLE_REDIRECT_URL must use HTTPS when Google OAuth is enabled in production")
	}
	if cfg.Environment == "production" {
		for name, databaseURL := range map[string]string{"DATABASE_URL": cfg.DatabaseURL, "MIGRATOR_DATABASE_URL": cfg.MigratorDatabaseURL, "IMPORTER_DATABASE_URL": cfg.ImporterDatabaseURL} {
			if databaseURL == "" {
				continue
			}
			databaseConfig, err := pgxpool.ParseConfig(databaseURL)
			if err != nil {
				return Config{}, errors.New(name + " is invalid")
			}
			if databaseConfig.ConnConfig.Config.TLSConfig == nil {
				return Config{}, errors.New(name + " must enable TLS when APP_ENV=production")
			}
			for _, fallback := range databaseConfig.ConnConfig.Config.Fallbacks {
				if fallback.TLSConfig == nil {
					return Config{}, errors.New(name + " must not allow plaintext connections when APP_ENV=production")
				}
			}
		}
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg, nil
}

func firstSet(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
