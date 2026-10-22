package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthAndReadiness(t *testing.T) {
	router := NewRouter(Dependencies{Ready: func(context.Context) error { return nil }})
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
	router := NewRouter(Dependencies{Ready: func(context.Context) error { return context.Canceled }})
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

func TestTimeoutReturnsStructuredJSON(t *testing.T) {
	handler := timeoutHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(20 * time.Millisecond) }), time.Millisecond)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["error"] != "request timeout" {
		t.Fatalf("body = %v", body)
	}
}
