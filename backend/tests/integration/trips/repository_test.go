package trips_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"tripplanner/backend/internal/db"
	"tripplanner/backend/internal/trips"
	"tripplanner/backend/tests/testutil"
)

func setupRepository(t *testing.T) (*testutil.PostGISDatabase, *trips.Repository) {
	t.Helper()
	testDB := testutil.StartPostGIS(t)
	testDB.CreateLegacySchema(t)
	if err := db.Apply(context.Background(), testDB.Pool, filepath.Join("..", "..", "..", "migrations")); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return testDB, trips.NewRepository(testDB.Pool)
}

func TestRepositoryRoundTripsTripsAndFiltersByOwner(t *testing.T) {
	testDB, repo := setupRepository(t)
	testDB.CreateSecondUser(t)
	ctx := context.Background()
	created, err := repo.CreateTrip(ctx, 41, trips.TripInput{Destination: "Bengaluru", DestinationID: 10, StartDate: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), DayCount: 2, Days: []trips.DayInput{{DayNumber: 1, AttractionIDs: []int64{101, 102}}, {DayNumber: 2, AttractionIDs: []int64{103}}}})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}
	if created.ID == 0 || created.UserID != 41 || created.Destination != "Bengaluru" || len(created.Days) != 2 {
		t.Fatalf("created trip=%#v", created)
	}
	if got := created.Days[0].Stops[1].AttractionID; got != 102 {
		t.Fatalf("second ordered stop=%d, want 102", got)
	}
	listed, err := repo.ListTrips(ctx, 41)
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("list trips=%#v, %v", listed, err)
	}
	if _, err := repo.GetTrip(ctx, 42, created.ID); !errors.Is(err, trips.ErrNotFound) {
		t.Fatalf("foreign get error=%v, want ErrNotFound", err)
	}
	updated, err := repo.UpdateTrip(ctx, 41, created.ID, trips.TripInput{Destination: "Bengaluru", StartDate: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), DayCount: 1, Days: []trips.DayInput{{DayNumber: 1, AttractionIDs: []int64{103, 101}}}})
	if err != nil {
		t.Fatalf("update trip: %v", err)
	}
	if updated.DayCount != 1 || len(updated.Days) != 1 || len(updated.Days[0].Stops) != 2 || updated.Days[0].Stops[0].AttractionID != 103 {
		t.Fatalf("updated trip=%#v", updated)
	}
	if err := repo.DeleteTrip(ctx, 42, created.ID); !errors.Is(err, trips.ErrNotFound) {
		t.Fatalf("foreign delete error=%v, want ErrNotFound", err)
	}
	if err := repo.DeleteTrip(ctx, 41, created.ID); err != nil {
		t.Fatalf("delete trip: %v", err)
	}
	if _, err := repo.GetTrip(ctx, 41, created.ID); !errors.Is(err, trips.ErrNotFound) {
		t.Fatalf("get deleted trip error=%v, want ErrNotFound", err)
	}
}

func TestUpdateRollsBackWhenAttractionDoesNotExist(t *testing.T) {
	_, repo := setupRepository(t)
	ctx := context.Background()
	created, err := repo.CreateTrip(ctx, 41, trips.TripInput{Destination: "Bengaluru", DestinationID: 10, StartDate: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), DayCount: 1, Days: []trips.DayInput{{DayNumber: 1, AttractionIDs: []int64{101}}}})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}
	_, err = repo.UpdateTrip(ctx, 41, created.ID, trips.TripInput{Destination: "Bengaluru", DestinationID: 10, StartDate: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), DayCount: 1, Days: []trips.DayInput{{DayNumber: 1, AttractionIDs: []int64{9999}}}})
	if err == nil {
		t.Fatal("update with unknown attraction succeeded")
	}
	after, err := repo.GetTrip(ctx, 41, created.ID)
	if err != nil {
		t.Fatalf("get trip after rollback: %v", err)
	}
	if !after.StartDate.Equal(created.StartDate) || after.Days[0].Stops[0].AttractionID != 101 {
		t.Fatalf("failed update changed trip: %#v", after)
	}
}
