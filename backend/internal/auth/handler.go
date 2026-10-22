package auth

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	CookieSecure  bool
	AllowedOrigin string
}
type Handler struct {
	pool          *pgxpool.Pool
	cookieSecure  bool
	allowedOrigin string
}
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func NewHandler(pool *pgxpool.Pool, options Options) *Handler {
	return &Handler{pool: pool, cookieSecure: options.CookieSecure, allowedOrigin: strings.TrimRight(options.AllowedOrigin, "/")}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.Handle("GET /api/v1/auth/me", h.RequireAuth(http.HandlerFunc(h.me)))
	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.checkOrigin(w, r) {
		return
	}
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = normalizeEmail(input.Email)
	if _, err := mail.ParseAddress(input.Email); err != nil || !strings.EqualFold(input.Email, strings.TrimSpace(input.Email)) || len(input.Email) > 254 {
		writeError(w, http.StatusBadRequest, "enter a valid email")
		return
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.SplitN(input.Email, "@", 2)[0]
	var user User
	err = h.pool.QueryRow(r.Context(), `INSERT INTO users (username,email,password,full_name) VALUES ($1,$2,$3,$4) RETURNING id,COALESCE(username,''),email,COALESCE(full_name,'')`, username, input.Email, hash, strings.TrimSpace(input.FullName)).Scan(&user.ID, &user.Username, &user.Email, &user.FullName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "an account with that email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	if err := h.issueSession(w, user); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if !h.checkOrigin(w, r) {
		return
	}
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	var user User
	var hash string
	err := h.pool.QueryRow(r.Context(), `SELECT id,COALESCE(username,''),email,COALESCE(full_name,''),password FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(input.Email)).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &hash)
	if err != nil || !VerifyPassword(hash, input.Password) {
		writeError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
		return
	}
	if err := h.issueSession(w, user); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if !h.checkOrigin(w, r) {
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		hash := sha256Bytes(cookie.Value)
		_, _ = h.pool.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", hash[:])
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, user)
}
func (h *Handler) checkOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	if h.allowedOrigin != "" && origin == h.allowedOrigin {
		return true
	}
	writeError(w, http.StatusForbidden, "request origin is not allowed")
	return false
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func sha256Bytes(value string) [32]byte { return sha256.Sum256([]byte(value)) }
