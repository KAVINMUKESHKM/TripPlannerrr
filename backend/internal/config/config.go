package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL        string
	OpenWeatherAPIKey  string
	OpenCageAPIKey     string
	GoogleMapsAPIKey   string
	GoogleClientID     string
	GoogleClientSecret string
	FrontendURL        string
	Port               string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"), OpenWeatherAPIKey: os.Getenv("OPENWEATHER_API_KEY"),
		OpenCageAPIKey: os.Getenv("OPENCAGE_API_KEY"), GoogleMapsAPIKey: os.Getenv("GOOGLE_MAPS_API_KEY"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		FrontendURL: os.Getenv("FRONTEND_URL"), Port: os.Getenv("PORT"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg, nil
}
