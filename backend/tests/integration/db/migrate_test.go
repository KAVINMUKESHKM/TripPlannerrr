package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"tripplanner/backend/internal/db"
	"tripplanner/backend/tests/testutil"
)

func TestApplyPreservesLegacyRowsAndIsRepeatable(t *testing.T) {
	testDB := testutil.StartPostGIS(t)
	testDB.CreateLegacySchema(t)
	ctx := context.Background()
	migrations := filepath.Join("..", "..", "..", "migrations")
	if err := db.Apply(ctx, testDB.Pool, migrations); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := db.Apply(ctx, testDB.Pool, migrations); err != nil {
		t.Fatalf("apply migrations twice: %v", err)
	}

	var userID, attractionID int
	var passwordHash, locationWKT string
	if err := testDB.Pool.QueryRow(ctx, "SELECT id, password FROM users WHERE email = 'legacy@example.com'").Scan(&userID, &passwordHash); err != nil {
		t.Fatal(err)
	}
	if userID != 41 || passwordHash != testutil.LegacyPasswordHash {
		t.Fatalf("legacy user changed: id=%d password=%q", userID, passwordHash)
	}
	if err := testDB.Pool.QueryRow(ctx, "SELECT id, ST_AsText(location) FROM attractions WHERE name = 'Legacy monument'").Scan(&attractionID, &locationWKT); err != nil {
		t.Fatal(err)
	}
	if attractionID != 101 || locationWKT != "POINT(77.5946 12.9716)" {
		t.Fatalf("legacy attraction changed: id=%d location=%q", attractionID, locationWKT)
	}
	for _, table := range []string{"sessions", "trips", "trip_days", "trip_stops", "api_rate_limits", "password_reset_tokens"} {
		var exists bool
		if err := testDB.Pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("%s was not created", table)
		}
	}
	var googleSubject *string
	if err := testDB.Pool.QueryRow(ctx, "SELECT google_subject FROM users WHERE id=41").Scan(&googleSubject); err != nil {
		t.Fatal(err)
	}
	if googleSubject != nil {
		t.Fatalf("google_subject=%q, want NULL", *googleSubject)
	}
}
