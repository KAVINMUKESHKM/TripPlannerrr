package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

const passwordResetLifetime = 30 * time.Minute

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !h.checkOrigin(w, r) {
		return
	}
	if !isPasswordResetConfigured(h.smtp, h.frontendURL) {
		writeError(w, http.StatusServiceUnavailable, "password reset email is not configured")
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	email := normalizeEmail(input.Email)
	var userID int64
	err := h.pool.QueryRow(r.Context(), "SELECT id FROM users WHERE lower(email)=lower($1)", email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusAccepted, map[string]string{"message": "If that account exists, password reset instructions will be sent."})
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create reset request")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(token))
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(r.Context(), "DELETE FROM password_reset_tokens WHERE user_id=$1", userID); err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	if _, err := tx.Exec(r.Context(), "INSERT INTO password_reset_tokens(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hash[:], userID, time.Now().Add(passwordResetLifetime)); err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	resetURL := h.frontendURL + "/reset-password?token=" + url.QueryEscape(token)
	mailContext, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	err = h.smtp.send(mailContext, email, "Reset your Trip Planner password", "Use this single-use link within 30 minutes to reset your password:\r\n\r\n"+resetURL+"\r\n\r\nIf you did not request this, you can ignore this email.")
	cancel()
	if err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
		_, _ = h.pool.Exec(cleanupCtx, "DELETE FROM password_reset_tokens WHERE token_hash=$1", hash[:])
		cleanupCancel()
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "If that account exists, password reset instructions will be sent."})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !h.checkOrigin(w, r) {
		return
	}
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Token == "" || len(input.Token) > 128 {
		writeError(w, http.StatusBadRequest, "reset link is invalid or expired")
		return
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tokenHash := sha256.Sum256([]byte(input.Token))
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "password reset is temporarily unavailable")
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var userID int64
	err = tx.QueryRow(r.Context(), `DELETE FROM password_reset_tokens WHERE token_hash=$1 AND expires_at > now() RETURNING user_id`, tokenHash[:]).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "reset link is invalid or expired")
		return
	}
	if _, err := tx.Exec(r.Context(), "UPDATE users SET password=$1 WHERE id=$2", hash, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset password")
		return
	}
	if _, err := tx.Exec(r.Context(), "DELETE FROM sessions WHERE user_id=$1", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset password")
		return
	}
	if _, err := tx.Exec(r.Context(), "DELETE FROM password_reset_tokens WHERE user_id=$1", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset password")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset password")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func isPasswordResetConfigured(config SMTPConfig, frontendURL string) bool {
	if !config.enabled() {
		return false
	}
	parsed, err := url.Parse(frontendURL)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(parsed.Hostname())
	return parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))
}
