package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

func HashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 72 {
		return "", errors.New("password must be between 8 and 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
