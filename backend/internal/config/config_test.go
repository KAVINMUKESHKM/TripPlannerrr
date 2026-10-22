package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() should reject a missing DATABASE_URL")
	}
}

func TestLoadReadsDatabaseURLAndOptionalProviderKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("OPENWEATHER_API_KEY", "weather-key")
	t.Setenv("OPENCAGE_API_KEY", "geo-key")
	t.Setenv("GOOGLE_MAPS_API_KEY", "maps-key")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://example" || cfg.OpenWeatherAPIKey != "weather-key" || cfg.OpenCageAPIKey != "geo-key" || cfg.GoogleMapsAPIKey != "maps-key" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
