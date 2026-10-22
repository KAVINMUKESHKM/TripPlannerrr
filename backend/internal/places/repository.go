package places

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tripplanner/backend/internal/providers"
)

var ErrNotFound = errors.New("attraction not found")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) FindDestination(ctx context.Context, name string) (providers.Coordinate, error) {
	var point providers.Coordinate
	err := r.pool.QueryRow(ctx, `SELECT latitude,longitude FROM destinations WHERE lower(name)=lower($1) LIMIT 1`, strings.TrimSpace(name)).Scan(&point.Latitude, &point.Longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		return providers.Coordinate{}, ErrNotFound
	}
	if err != nil {
		return providers.Coordinate{}, fmt.Errorf("find destination: %w", err)
	}
	if point.Latitude < -90 || point.Latitude > 90 || point.Longitude < -180 || point.Longitude > 180 {
		return providers.Coordinate{}, errors.New("destination coordinates are invalid")
	}
	return point, nil
}

func (r *Repository) Search(ctx context.Context, point providers.Coordinate, radiusKM float64, category string) ([]Attraction, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,COALESCE(name,''),COALESCE(description,''),COALESCE(category,''),COALESCE(latitude,ST_Y(location::geometry)),COALESCE(longitude,ST_X(location::geometry))
		FROM attractions WHERE location IS NOT NULL AND ST_DWithin(location::geography,ST_SetSRID(ST_MakePoint($1,$2),4326)::geography,$3)
		AND ($4='' OR lower(category)=lower($4)) ORDER BY id LIMIT 200`, point.Longitude, point.Latitude, radiusKM*1000, strings.TrimSpace(category))
	if err != nil {
		return nil, fmt.Errorf("search attractions: %w", err)
	}
	defer rows.Close()
	places := make([]Attraction, 0)
	for rows.Next() {
		var place Attraction
		if err := rows.Scan(&place.ID, &place.Name, &place.Description, &place.Category, &place.Latitude, &place.Longitude); err != nil {
			return nil, fmt.Errorf("read attraction: %w", err)
		}
		places = append(places, place)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read attractions: %w", err)
	}
	return places, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Attraction, error) {
	var place Attraction
	err := r.pool.QueryRow(ctx, `SELECT id,COALESCE(name,''),COALESCE(description,''),COALESCE(category,''),COALESCE(latitude,ST_Y(location::geometry)),COALESCE(longitude,ST_X(location::geometry)) FROM attractions WHERE id=$1`, id).Scan(&place.ID, &place.Name, &place.Description, &place.Category, &place.Latitude, &place.Longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attraction{}, ErrNotFound
	}
	if err != nil {
		return Attraction{}, fmt.Errorf("get attraction: %w", err)
	}
	return place, nil
}
