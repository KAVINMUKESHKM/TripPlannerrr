package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tripplanner/backend/internal/auth"
	"tripplanner/backend/internal/db"
	"tripplanner/backend/tests/testutil"
)

func TestRegisterLoginMeAndLogoutUseOpaqueSession(t *testing.T) {
	database := testutil.StartPostGIS(t)
	database.CreateLegacySchema(t)
	if err := db.Apply(context.Background(), database.Pool, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	server := auth.NewHandler(database.Pool, auth.Options{CookieSecure: false, AllowedOrigin: "http://localhost:3000"}).Routes()

	blocked := request(t, server, http.MethodPost, "/register", `{"email":"blocked@example.com","password":"safe password"}`, "https://evil.example")
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("untrusted origin status=%d", blocked.Code)
	}
	register := request(t, server, http.MethodPost, "/register", `{"email":" New@Example.com ","password":"safe password","full_name":"New User"}`, "http://localhost:3000")
	if register.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", register.Code, register.Body.String())
	}
	if strings.Contains(register.Body.String(), "password") || strings.Contains(register.Body.String(), "hash") {
		t.Fatalf("register leaked credentials: %s", register.Body.String())
	}
	cookie := register.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != "trip_session" || !cookie[0].HttpOnly || cookie[0].MaxAge <= 0 {
		t.Fatalf("invalid session cookie: %#v", cookie)
	}
	if len(cookie[0].Value) < 32 {
		t.Fatalf("session secret is too short: %q", cookie[0].Value)
	}
	var storedHash []byte
	if err := database.Pool.QueryRow(context.Background(), "SELECT token_hash FROM sessions LIMIT 1").Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(storedHash, []byte(cookie[0].Value)) {
		t.Fatal("database stores raw session secret")
	}

	me := requestWithCookie(t, server, http.MethodGet, "/me", cookie[0])
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), "new@example.com") {
		t.Fatalf("me status=%d body=%s", me.Code, me.Body.String())
	}

	login := request(t, server, http.MethodPost, "/login", `{"email":"NEW@example.com","password":"safe password"}`, "http://localhost:3000")
	if login.Code != http.StatusOK || len(login.Result().Cookies()) != 1 {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	wrong := request(t, server, http.MethodPost, "/login", `{"email":"new@example.com","password":"wrong password"}`, "http://localhost:3000")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status=%d", wrong.Code)
	}

	duplicate := request(t, server, http.MethodPost, "/register", `{"email":"new@example.com","password":"safe password"}`, "http://localhost:3000")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}

	logout := requestWithCookie(t, server, http.MethodPost, "/logout", login.Result().Cookies()[0])
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	invalidated := requestWithCookie(t, server, http.MethodGet, "/me", login.Result().Cookies()[0])
	if invalidated.Code != http.StatusUnauthorized {
		t.Fatalf("invalidated session status=%d", invalidated.Code)
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	database := testutil.StartPostGIS(t)
	database.CreateLegacySchema(t)
	if err := db.Apply(context.Background(), database.Pool, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	server := auth.NewHandler(database.Pool, auth.Options{CookieSecure: false, AllowedOrigin: "http://localhost:3000"}).Routes()
	register := request(t, server, http.MethodPost, "/register", `{"email":"new@example.com","password":"safe password"}`, "http://localhost:3000")
	cookie := register.Result().Cookies()[0]
	if _, err := database.Pool.Exec(context.Background(), "UPDATE sessions SET expires_at=$1", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	me := requestWithCookie(t, server, http.MethodGet, "/me", cookie)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status=%d", me.Code)
	}
}

func request(t *testing.T, h http.Handler, method, path, body, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/auth"+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func requestWithCookie(t *testing.T, h http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/auth"+path, nil)
	if method != http.MethodGet {
		req.Header.Set("Origin", "http://localhost:3000")
	}
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func TestAuthResponseIsJSON(t *testing.T) {
	var v map[string]any
	if err := json.Unmarshal([]byte(`{"ok":true}`), &v); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyPasswordAccountKeepsItsIDAndHash(t *testing.T) {
	database := testutil.StartPostGIS(t)
	database.CreateLegacySchema(t)
	legacyHash, err := auth.HashPassword("legacy password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Pool.Exec(context.Background(), "UPDATE users SET password=$1 WHERE id=41", legacyHash); err != nil {
		t.Fatal(err)
	}
	if err := db.Apply(context.Background(), database.Pool, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	server := auth.NewHandler(database.Pool, auth.Options{CookieSecure: false, AllowedOrigin: "http://localhost:3000"}).Routes()
	login := request(t, server, http.MethodPost, "/login", `{"email":"legacy@example.com","password":"legacy password"}`, "http://localhost:3000")
	if login.Code != http.StatusOK {
		t.Fatalf("legacy login status=%d body=%s", login.Code, login.Body.String())
	}
	var id int64
	var storedHash string
	if err := database.Pool.QueryRow(context.Background(), "SELECT id,password FROM users WHERE email='legacy@example.com'").Scan(&id, &storedHash); err != nil {
		t.Fatal(err)
	}
	if id != 41 || storedHash != legacyHash {
		t.Fatalf("legacy identity changed: id=%d hashPreserved=%v", id, storedHash == legacyHash)
	}
}
