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
	t.Setenv("OSRM_BASE_URL", "https://router.project-osrm.org")
	t.Setenv("OPENFREEMAP_STYLE_URL", "https://tiles.openfreemap.org/styles/liberty")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://example" || cfg.OpenWeatherAPIKey != "weather-key" || cfg.OpenCageAPIKey != "geo-key" || cfg.OSRMBaseURL != "https://router.project-osrm.org" || cfg.OpenFreeMapStyleURL != "https://tiles.openfreemap.org/styles/liberty" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
