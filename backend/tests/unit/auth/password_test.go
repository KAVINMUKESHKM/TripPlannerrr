package auth_test

import (
	"testing"

	"tripplanner/backend/internal/auth"
)

func TestHashPasswordRejectsShortAndOverlongInputs(t *testing.T) {
	for _, password := range []string{"short", string(make([]byte, 73))} {
		if _, err := auth.HashPassword(password); err == nil {
			t.Fatal("HashPassword accepted an unsupported password length")
		}
	}
}

func TestHashPasswordVerifiesWithoutStoringThePlaintext(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal("HashPassword returned an error")
	}
	if hash == password || !auth.VerifyPassword(hash, password) || auth.VerifyPassword(hash, "incorrect password") {
		t.Fatal("password hash verification behaved incorrectly")
	}
}
