package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
)

const oauthStateLifetime = 15 * time.Minute

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

type IdentityVerifier interface {
	Verify(context.Context, string, string) (GoogleIdentity, error)
}

type GoogleOIDCVerifier struct{ keySet oidc.KeySet }

func NewGoogleOIDCVerifier() *GoogleOIDCVerifier {
	return &GoogleOIDCVerifier{keySet: oidc.NewRemoteKeySet(context.Background(), "https://www.googleapis.com/oauth2/v3/certs")}
}

func (v *GoogleOIDCVerifier) Verify(ctx context.Context, rawToken, audience string) (GoogleIdentity, error) {
	if v == nil || v.keySet == nil {
		return GoogleIdentity{}, errors.New("Google identity verification is not configured")
	}
	verifier := oidc.NewVerifier("https://accounts.google.com", v.keySet, &oidc.Config{ClientID: audience})
	token, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return GoogleIdentity{}, errors.New("Google identity token is invalid")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return GoogleIdentity{}, errors.New("Google identity claims are invalid")
	}
	return GoogleIdentity{Subject: token.Subject, Email: normalizeEmail(claims.Email), EmailVerified: claims.EmailVerified, Name: strings.TrimSpace(claims.Name)}, nil
}

func (h *Handler) googleStart(w http.ResponseWriter, r *http.Request) {
	if h.googleOAuth == nil || h.googleVerifier == nil || h.frontendURL == "" {
		writeError(w, http.StatusServiceUnavailable, "Google sign-in is not configured")
		return
	}
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start Google sign-in")
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	hash := sha256.Sum256([]byte(state))
	if _, err := h.pool.Exec(r.Context(), "INSERT INTO oauth_states(state_hash,expires_at) VALUES($1,$2)", hash[:], time.Now().Add(oauthStateLifetime)); err != nil {
		writeError(w, http.StatusServiceUnavailable, "Google sign-in is temporarily unavailable")
		return
	}
	http.Redirect(w, r, h.googleOAuth.AuthCodeURL(state, oauth2.SetAuthURLParam("prompt", "select_account")), http.StatusFound)
}

func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	if h.googleOAuth == nil || h.googleVerifier == nil || h.frontendURL == "" {
		writeError(w, http.StatusServiceUnavailable, "Google sign-in is not configured")
		return
	}
	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, h.frontendURL+"/auth/callback?error=cancelled", http.StatusFound)
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" || len(state) > 256 || !h.consumeOAuthState(r.Context(), state) {
		writeError(w, http.StatusBadRequest, "Google sign-in state is invalid or expired")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "Google did not return an authorization code")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	token, err := h.googleOAuth.Exchange(ctx, code)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Google sign-in could not be completed")
		return
	}
	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		writeError(w, http.StatusUnauthorized, "Google did not return an identity token")
		return
	}
	identity, err := h.googleVerifier.Verify(ctx, rawIDToken, h.googleOAuth.ClientID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Google identity could not be verified")
		return
	}
	if identity.Subject == "" || identity.Email == "" || !identity.EmailVerified {
		writeError(w, http.StatusForbidden, "Google must verify your email before sign-in")
		return
	}
	user, err := h.resolveGoogleUser(ctx, identity)
	if err != nil {
		writeError(w, http.StatusConflict, "Google account could not be linked to a user")
		return
	}
	if err := h.issueSession(w, user); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	http.Redirect(w, r, h.frontendURL+"/auth/callback", http.StatusFound)
}

func (h *Handler) consumeOAuthState(ctx context.Context, state string) bool {
	hash := sha256.Sum256([]byte(state))
	_, _ = h.pool.Exec(ctx, "DELETE FROM oauth_states WHERE expires_at <= now()")
	var consumed []byte
	err := h.pool.QueryRow(ctx, "DELETE FROM oauth_states WHERE state_hash=$1 AND expires_at > now() RETURNING state_hash", hash[:]).Scan(&consumed)
	return err == nil
}

func (h *Handler) resolveGoogleUser(ctx context.Context, identity GoogleIdentity) (User, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var user User
	err = tx.QueryRow(ctx, `SELECT id,COALESCE(username,''),email,COALESCE(full_name,'') FROM users WHERE google_subject=$1`, identity.Subject).Scan(&user.ID, &user.Username, &user.Email, &user.FullName)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return User{}, err
		}
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, fmt.Errorf("find Google subject: %w", err)
	}
	err = tx.QueryRow(ctx, `SELECT id,COALESCE(username,''),email,COALESCE(full_name,'') FROM users WHERE lower(email)=lower($1)`, normalizeEmail(identity.Email)).Scan(&user.ID, &user.Username, &user.Email, &user.FullName)
	if err == nil {
		var linkedSubject pgtype.Text
		if err = tx.QueryRow(ctx, "UPDATE users SET google_subject=$1 WHERE id=$2 AND google_subject IS NULL RETURNING google_subject", identity.Subject, user.ID).Scan(&linkedSubject); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return User{}, err
			}
			if err = tx.QueryRow(ctx, "SELECT google_subject FROM users WHERE id=$1", user.ID).Scan(&linkedSubject); err != nil || !linkedSubject.Valid || linkedSubject.String != identity.Subject {
				return User{}, errors.New("password account is already linked to another Google identity")
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, err
		}
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, fmt.Errorf("find Google email: %w", err)
	}
	email := normalizeEmail(identity.Email)
	username := strings.SplitN(email, "@", 2)[0]
	randomPassword := make([]byte, 32)
	if _, err := rand.Read(randomPassword); err != nil {
		return User{}, err
	}
	passwordHash, err := HashPassword(base64.RawURLEncoding.EncodeToString(randomPassword))
	if err != nil {
		return User{}, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO users(username,email,password,full_name,google_subject) VALUES($1,$2,$3,$4,$5) RETURNING id,COALESCE(username,''),email,COALESCE(full_name,'')`, username, email, passwordHash, identity.Name, identity.Subject).Scan(&user.ID, &user.Username, &user.Email, &user.FullName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, errors.New("Google account conflicts with an existing user")
		}
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}
