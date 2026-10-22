package trips

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) CreateTrip(ctx context.Context, userID int64, input TripInput) (Trip, error) {
	if err := validateInput(input); err != nil {
		return Trip{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Trip{}, fmt.Errorf("begin create trip: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO trips (user_id, destination_id, destination_name, start_date, day_count)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, userID, nullableID(input.DestinationID), strings.TrimSpace(input.Destination), input.StartDate, input.DayCount).Scan(&id); err != nil {
		return Trip{}, fmt.Errorf("insert trip: %w", err)
	}
	if err := writeDays(ctx, tx, id, input); err != nil {
		return Trip{}, err
	}
	trip, err := loadTrip(ctx, tx, userID, id)
	if err != nil {
		return Trip{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Trip{}, fmt.Errorf("commit trip: %w", err)
	}
	return trip, nil
}

func (r *Repository) ListTrips(ctx context.Context, userID int64) ([]Trip, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, destination_id, destination_name, start_date, day_count, created_at, updated_at
		FROM trips WHERE user_id = $1 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list trips: %w", err)
	}
	trips := make([]Trip, 0)
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan trip: %w", err)
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read trips: %w", err)
	}
	rows.Close()
	for i := range trips {
		trips[i].Days, err = loadDays(ctx, r.pool, trips[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return trips, nil
}

func (r *Repository) GetTrip(ctx context.Context, userID, tripID int64) (Trip, error) {
	return loadTrip(ctx, r.pool, userID, tripID)
}

func (r *Repository) UpdateTrip(ctx context.Context, userID, tripID int64, input TripInput) (Trip, error) {
	if err := validateInput(input); err != nil {
		return Trip{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Trip{}, fmt.Errorf("begin update trip: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	result, err := tx.Exec(ctx, `UPDATE trips SET destination_id = $1, destination_name = $2, start_date = $3,
		day_count = $4, updated_at = now() WHERE id = $5 AND user_id = $6`, nullableID(input.DestinationID), strings.TrimSpace(input.Destination), input.StartDate, input.DayCount, tripID, userID)
	if err != nil {
		return Trip{}, fmt.Errorf("update trip: %w", err)
	}
	if result.RowsAffected() == 0 {
		return Trip{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, "DELETE FROM trip_days WHERE trip_id = $1", tripID); err != nil {
		return Trip{}, fmt.Errorf("replace trip days: %w", err)
	}
	if err := writeDays(ctx, tx, tripID, input); err != nil {
		return Trip{}, err
	}
	trip, err := loadTrip(ctx, tx, userID, tripID)
	if err != nil {
		return Trip{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Trip{}, fmt.Errorf("commit trip update: %w", err)
	}
	return trip, nil
}

func (r *Repository) DeleteTrip(ctx context.Context, userID, tripID int64) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM trips WHERE id = $1 AND user_id = $2", tripID, userID)
	if err != nil {
		return fmt.Errorf("delete trip: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func validateInput(input TripInput) error {
	if strings.TrimSpace(input.Destination) == "" {
		return errors.New("destination is required")
	}
	if input.DestinationID < 0 {
		return errors.New("destination ID must be positive")
	}
	if input.DayCount < 1 || input.DayCount > 30 {
		return errors.New("day count must be between 1 and 30")
	}
	if input.StartDate.IsZero() {
		return errors.New("start date is required")
	}
	seenDays := make(map[int]struct{}, len(input.Days))
	for _, day := range input.Days {
		if day.DayNumber < 1 || day.DayNumber > input.DayCount {
			return fmt.Errorf("day number %d is outside the trip", day.DayNumber)
		}
		if _, exists := seenDays[day.DayNumber]; exists {
			return fmt.Errorf("day %d is duplicated", day.DayNumber)
		}
		seenDays[day.DayNumber] = struct{}{}
		seenAttractions := make(map[int64]struct{}, len(day.AttractionIDs))
		for _, attractionID := range day.AttractionIDs {
			if attractionID < 1 {
				return errors.New("attraction ID must be positive")
			}
			if _, exists := seenAttractions[attractionID]; exists {
				return fmt.Errorf("attraction %d is duplicated on day %d", attractionID, day.DayNumber)
			}
			seenAttractions[attractionID] = struct{}{}
		}
	}
	return nil
}

func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func writeDays(ctx context.Context, tx pgx.Tx, tripID int64, input TripInput) error {
	dayStops := make(map[int][]int64, len(input.Days))
	for _, day := range input.Days {
		dayStops[day.DayNumber] = day.AttractionIDs
	}
	for dayNumber := 1; dayNumber <= input.DayCount; dayNumber++ {
		if _, err := tx.Exec(ctx, "INSERT INTO trip_days (trip_id, day_number) VALUES ($1, $2)", tripID, dayNumber); err != nil {
			return fmt.Errorf("insert trip day %d: %w", dayNumber, err)
		}
		for position, attractionID := range dayStops[dayNumber] {
			if _, err := tx.Exec(ctx, `INSERT INTO trip_stops (trip_id, day_number, stop_position, attraction_id)
				VALUES ($1, $2, $3, $4)`, tripID, dayNumber, position, attractionID); err != nil {
				return fmt.Errorf("insert attraction %d on day %d: %w", attractionID, dayNumber, err)
			}
		}
	}
	return nil
}

func loadTrip(ctx context.Context, q queryer, userID, tripID int64) (Trip, error) {
	trip, err := scanTrip(q.QueryRow(ctx, `SELECT id, user_id, destination_id, destination_name, start_date, day_count, created_at, updated_at
		FROM trips WHERE user_id = $1 AND id = $2`, userID, tripID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, ErrNotFound
	}
	if err != nil {
		return Trip{}, fmt.Errorf("get trip: %w", err)
	}
	trip.Days, err = loadDays(ctx, q, tripID)
	if err != nil {
		return Trip{}, err
	}
	return trip, nil
}

func scanTrip(row pgx.Row) (Trip, error) {
	var trip Trip
	var destinationID pgtype.Int8
	var dayCount int16
	if err := row.Scan(&trip.ID, &trip.UserID, &destinationID, &trip.Destination, &trip.StartDate, &dayCount, &trip.CreatedAt, &trip.UpdatedAt); err != nil {
		return Trip{}, err
	}
	trip.DayCount = int(dayCount)
	if destinationID.Valid {
		trip.DestinationID = &destinationID.Int64
	}
	return trip, nil
}

func loadDays(ctx context.Context, q queryer, tripID int64) ([]Day, error) {
	rows, err := q.Query(ctx, `SELECT day_number, stop_position, attraction_id
		FROM trip_days d LEFT JOIN trip_stops s USING (trip_id, day_number)
		WHERE d.trip_id = $1 ORDER BY d.day_number, s.stop_position`, tripID)
	if err != nil {
		return nil, fmt.Errorf("load trip days: %w", err)
	}
	days := make([]Day, 0)
	for rows.Next() {
		var dayNumber int16
		var position, attractionID pgtype.Int8
		if err := rows.Scan(&dayNumber, &position, &attractionID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan trip day: %w", err)
		}
		if len(days) == 0 || days[len(days)-1].DayNumber != int(dayNumber) {
			days = append(days, Day{DayNumber: int(dayNumber), Stops: make([]Stop, 0)})
		}
		if attractionID.Valid {
			days[len(days)-1].Stops = append(days[len(days)-1].Stops, Stop{Position: int(position.Int64), AttractionID: attractionID.Int64})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read trip days: %w", err)
	}
	rows.Close()
	return days, nil
}

var _ queryer = (*pgxpool.Pool)(nil)
