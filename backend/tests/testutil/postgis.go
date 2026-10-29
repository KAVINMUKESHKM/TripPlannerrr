package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const LegacyPasswordHash = "$2b$12$abcdefghijklmnopqrstuuqj7v9ob2jR7Z1x5iVY0Zl4a9q1QdGzq"

type PostGISDatabase struct{ Pool *pgxpool.Pool }

func StartPostGIS(t testing.TB) *PostGISDatabase {
	t.Helper()
	if testing.Short() {
		t.Skip("PostGIS integration test")
	}
	image := os.Getenv("POSTGIS_TEST_IMAGE")
	if image == "" {
		image = "postgis/postgis:17-3.5"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	output, err := exec.CommandContext(ctx, "docker", "run", "--rm", "-d", "-e", "POSTGRES_PASSWORD=tripplanner", "-e", "POSTGRES_DB=tripplanner", "-p", "127.0.0.1::5432", image).Output()
	if err != nil {
		t.Fatalf("start PostGIS container: %v: %s", err, output)
	}
	containerID := strings.TrimSpace(string(output))
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", containerID).Run() })
	portOutput, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{(index (index .NetworkSettings.Ports \"5432/tcp\") 0).HostPort}}", containerID).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect PostGIS port: %v: %s", err, portOutput)
	}
	dsn := fmt.Sprintf("postgres://postgres:tripplanner@127.0.0.1:%s/tripplanner?sslmode=disable", strings.TrimSpace(string(portOutput)))
	deadline := time.Now().Add(45 * time.Second)
	for {
		pool, poolErr := pgxpool.New(context.Background(), dsn)
		if poolErr == nil {
			pingCtx, pingCancel := context.WithTimeout(context.Background(), time.Second)
			pingErr := pool.Ping(pingCtx)
			pingCancel()
			if pingErr == nil {
				t.Cleanup(pool.Close)
				return &PostGISDatabase{Pool: pool}
			}
			pool.Close()
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect to PostGIS: %v", poolErr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (db *PostGISDatabase) CreateLegacySchema(t testing.TB) {
	t.Helper()
	ctx := context.Background()
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS postgis`,
		`CREATE TABLE users (id SERIAL PRIMARY KEY, username VARCHAR(255), email VARCHAR(255) NOT NULL UNIQUE, password TEXT NOT NULL, full_name TEXT, bio TEXT)`,
		`CREATE TABLE destinations (id SERIAL PRIMARY KEY, name VARCHAR(255), description TEXT, latitude DECIMAL(9,6), longitude DECIMAL(9,6))`,
		`CREATE TABLE attractions (id SERIAL PRIMARY KEY, name VARCHAR(255), description TEXT, category VARCHAR(100), latitude DECIMAL(9,6), longitude DECIMAL(9,6), destination_id INTEGER REFERENCES destinations(id), location GEOGRAPHY(Point, 4326))`,
		`INSERT INTO users (id, username, email, password, full_name, bio) VALUES (41, 'legacy', 'legacy@example.com', $1, 'Legacy User', 'preserve me')`,
		`INSERT INTO destinations (id, name, description, latitude, longitude) VALUES (10, 'Bengaluru', 'Legacy destination', 12.971600, 77.594600)`,
		`INSERT INTO attractions (id, name, description, category, latitude, longitude, destination_id, location) VALUES (101, 'Legacy monument', 'Preserved row', 'History', 12.971600, 77.594600, 10, ST_SetSRID(ST_MakePoint(77.594600, 12.971600), 4326)::geography), (102, 'Legacy garden', 'Preserved row', 'Nature', 12.980000, 77.590000, 10, ST_SetSRID(ST_MakePoint(77.590000, 12.980000), 4326)::geography), (103, 'Legacy museum', 'Preserved row', 'Culture', 12.975000, 77.600000, 10, ST_SetSRID(ST_MakePoint(77.600000, 12.975000), 4326)::geography)`,
	}
	for i, statement := range statements {
		var err error
		if i == 4 {
			_, err = db.Pool.Exec(ctx, statement, LegacyPasswordHash)
		} else {
			_, err = db.Pool.Exec(ctx, statement)
		}
		if err != nil {
			t.Fatalf("create legacy schema statement %d: %v", i+1, err)
		}
	}
}

func (db *PostGISDatabase) CreateSecondUser(t testing.TB) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), "INSERT INTO users (id, username, email, password) VALUES (42, 'other', 'other@example.com', $1)", LegacyPasswordHash)
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
}
