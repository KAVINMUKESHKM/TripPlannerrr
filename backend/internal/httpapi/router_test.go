package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
	if body := response.Body.String(); body == "" || body == "postgres" || body == "DATABASE_URL" {
		t.Fatalf("readiness response may expose a secret: %q", body)
	}
}
