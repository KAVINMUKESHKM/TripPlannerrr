package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"tripplanner/backend/internal/auth"
	"tripplanner/backend/internal/db"
	"tripplanner/backend/internal/testutil"
)

type fixedIdentityVerifier struct {
	identity auth.GoogleIdentity
	err      error
}

func (v fixedIdentityVerifier) Verify(context.Context, string, string) (auth.GoogleIdentity, error) {
	return v.identity, v.err
}

func TestGoogleOAuthCreatesAccountSessionAndConsumesStateOnce(t *testing.T) {
	database := testutil.StartPostGIS(t)
	database.CreateLegacySchema(t)
	if err := db.Apply(context.Background(), database.Pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("code") != "test-code" {
			t.Errorf("code=%q", r.Form.Get("code"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access", "token_type": "Bearer", "id_token": "signed-test-token", "expires_in": 3600})
	}))
	defer tokenServer.Close()
	oauthConfig := &oauth2.Config{ClientID: "client-id", ClientSecret: "client-secret", RedirectURL: "http://api.test/api/v1/auth/google/callback", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.test/auth", TokenURL: tokenServer.URL}}
	server := auth.NewHandler(database.Pool, auth.Options{CookieSecure: false, AllowedOrigin: "http://localhost:3000", GoogleOAuth: oauthConfig, GoogleVerifier: fixedIdentityVerifier{identity: auth.GoogleIdentity{Subject: "google-subject-1", Email: "Traveler@Example.com", EmailVerified: true, Name: "Traveler"}}, FrontendURL: "http://localhost:3000"}).Routes()
	start := httptest.NewRecorder()
	server.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil))
	if start.Code != http.StatusFound {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	redirect, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := redirect.Query().Get("state")
	if len(state) < 32 {
		t.Fatalf("weak OAuth state %q", state)
	}
	callback := httptest.NewRecorder()
	server.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state="+url.QueryEscape(state)+"&code=test-code", nil))
	if callback.Code != http.StatusFound || !strings.HasPrefix(callback.Header().Get("Location"), "http://localhost:3000/auth/callback") {
		t.Fatalf("callback status=%d location=%q body=%s", callback.Code, callback.Header().Get("Location"), callback.Body.String())
	}
	if len(callback.Result().Cookies()) != 1 || callback.Result().Cookies()[0].Name != "trip_session" {
		t.Fatalf("OAuth callback did not issue session cookie: %#v", callback.Result().Cookies())
	}
	var userID int64
	var email, subject string
	if err := database.Pool.QueryRow(context.Background(), "SELECT id,email,google_subject FROM users WHERE email='traveler@example.com'").Scan(&userID, &email, &subject); err != nil {
		t.Fatal(err)
	}
	if email != "traveler@example.com" || subject != "google-subject-1" {
		t.Fatalf("unexpected linked identity id=%d email=%q subject=%q", userID, email, subject)
	}
	replay := httptest.NewRecorder()
	server.ServeHTTP(replay, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state="+url.QueryEscape(state)+"&code=test-code", nil))
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replayed OAuth state status=%d body=%s", replay.Code, replay.Body.String())
	}
}

func TestGoogleOAuthRejectsUnverifiedEmail(t *testing.T) {
	database := testutil.StartPostGIS(t)
	database.CreateLegacySchema(t)
	if err := db.Apply(context.Background(), database.Pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access", "token_type": "Bearer", "id_token": "signed-test-token"})
	}))
	defer tokenServer.Close()
	server := auth.NewHandler(database.Pool, auth.Options{GoogleOAuth: &oauth2.Config{ClientID: "client-id", ClientSecret: "client-secret", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.test/auth", TokenURL: tokenServer.URL}}, GoogleVerifier: fixedIdentityVerifier{identity: auth.GoogleIdentity{Subject: "unverified-subject", Email: "unverified@example.com", EmailVerified: false}}, FrontendURL: "http://localhost:3000"}).Routes()
	start := httptest.NewRecorder()
	server.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil))
	redirect, _ := url.Parse(start.Header().Get("Location"))
	state := redirect.Query().Get("state")
	callback := httptest.NewRecorder()
	server.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state="+url.QueryEscape(state)+"&code=test-code", nil))
	if callback.Code != http.StatusForbidden {
		t.Fatalf("unverified callback status=%d body=%s", callback.Code, callback.Body.String())
	}
	var count int
	if err := database.Pool.QueryRow(context.Background(), "SELECT count(*) FROM users WHERE email='unverified@example.com'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("unverified Google identity created a user")
	}
}
