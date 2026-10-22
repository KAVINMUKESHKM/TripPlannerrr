package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"
)

const sessionCookieName = "trip_session"
const sessionLifetime = 30 * 24 * time.Hour

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email"`
	FullName string `json:"full_name,omitempty"`
}

type userContextKey struct{}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey{}).(User)
	return user, ok
}

func (h *Handler) issueSession(w http.ResponseWriter, user User) error {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(token))
	expires := time.Now().Add(sessionLifetime)
	if _, err := h.pool.Exec(context.Background(), "INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,$2,$3)", hash[:], user.ID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(sessionLifetime.Seconds())})
	return nil
}

func (h *Handler) authenticatedUser(r *http.Request) (User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return User{}, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	var user User
	err = h.pool.QueryRow(r.Context(), `SELECT u.id, COALESCE(u.username,''), u.email, COALESCE(u.full_name,'')
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at > now()`, hash[:]).Scan(&user.ID, &user.Username, &user.Email, &user.FullName)
	if err != nil {
		return User{}, ErrUnauthorized
	}
	return user, nil
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.authenticatedUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}

var ErrUnauthorized = errors.New("unauthorized")
