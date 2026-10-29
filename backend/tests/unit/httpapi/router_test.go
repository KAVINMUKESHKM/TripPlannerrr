package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tripplanner/backend/internal/httpapi"
)

func TestHealthAndReadiness(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{Ready: func(context.Context) error { return nil }})
	for _, path := range []string{"/api/v1/healthz", "/api/v1/readyz"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, response.Code)
		}
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("GET %s invalid JSON: %v", path, err)
		}
		if body["status"] != "ok" {
			t.Fatalf("GET %s body = %v", path, body)
		}
	}
}

func TestReadinessDoesNotExposeSecrets(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{Ready: func(context.Context) error { return context.Canceled }})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type = %q", contentType)
	}
	rawBody := response.Body.String()
	var body map[string]string
	if err := json.Unmarshal([]byte(rawBody), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "unavailable" || len(body) != 1 {
		t.Fatalf("unexpected readiness body: %v", body)
	}
	for _, secret := range []string{"postgres://", "DATABASE_URL", "OPENWEATHER_API_KEY", "GOOGLE_CLIENT_SECRET", "weather-key", "maps-key"} {
		if strings.Contains(rawBody, secret) {
			t.Fatalf("readiness response contains %q", secret)
		}
	}
}
