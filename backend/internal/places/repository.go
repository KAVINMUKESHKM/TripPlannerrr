package places

import (
	"context"
	"database/sql"
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
	rows, err := r.pool.Query(ctx, `WITH candidates AS (
		SELECT id,COALESCE(name,'') AS name,COALESCE(description,'') AS description,COALESCE(category,'') AS category,
			latitude::double precision AS latitude,longitude::double precision AS longitude,COALESCE(source_url,'') AS source_url,
			COALESCE(address,'') AS address,COALESCE(opening_hours,'') AS opening_hours,COALESCE(wheelchair,'') AS wheelchair,
			COALESCE(website,'') AS website,COALESCE(phone,'') AS phone,COALESCE(fee,'') AS fee,COALESCE(toilets,'') AS toilets,
			COALESCE(toilets_wheelchair,'') AS toilets_wheelchair,COALESCE(internet_access,'') AS internet_access,
			COALESCE(wikipedia,'') AS wikipedia,COALESCE(wikidata,'') AS wikidata,details_updated_at,
			COALESCE(best_climate,'') AS best_climate,COALESCE(ideal_weather,'') AS ideal_weather,ideal_temp_min,ideal_temp_max,
			6371.0088 * 2 * asin(sqrt(LEAST(1.0,
				power(sin(radians(latitude::double precision - $2) / 2), 2) +
				cos(radians($2)) * cos(radians(latitude::double precision)) *
				power(sin(radians(longitude::double precision - $1) / 2), 2)
			))) AS distance_km
		FROM attractions
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
			AND ($4='' OR lower(category)=lower($4))
	)
	SELECT id,name,description,category,latitude,longitude,source_url,address,opening_hours,wheelchair,website,phone,fee,toilets,toilets_wheelchair,internet_access,wikipedia,wikidata,details_updated_at,best_climate,ideal_weather,ideal_temp_min,ideal_temp_max FROM candidates
	WHERE distance_km <= $3 ORDER BY distance_km,id LIMIT 200`, point.Longitude, point.Latitude, radiusKM, strings.TrimSpace(category))
	if err != nil {
		return nil, fmt.Errorf("search attractions: %w", err)
	}
	defer rows.Close()
	places := make([]Attraction, 0)
	for rows.Next() {
		var place Attraction
		var idealTempMin, idealTempMax sql.NullFloat64
		var detailsUpdatedAt sql.NullTime
		if err := rows.Scan(&place.ID, &place.Name, &place.Description, &place.Category, &place.Latitude, &place.Longitude, &place.SourceURL, &place.Address, &place.OpeningHours, &place.Wheelchair, &place.Website, &place.Phone, &place.Fee, &place.Toilets, &place.ToiletsWheelchair, &place.InternetAccess, &place.Wikipedia, &place.Wikidata, &detailsUpdatedAt, &place.BestClimate, &place.IdealWeather, &idealTempMin, &idealTempMax); err != nil {
			return nil, fmt.Errorf("read attraction: %w", err)
		}
		if idealTempMin.Valid {
			place.IdealTempMin = &idealTempMin.Float64
		}
		if idealTempMax.Valid {
			place.IdealTempMax = &idealTempMax.Float64
		}
		if detailsUpdatedAt.Valid {
			place.DetailsUpdatedAt = &detailsUpdatedAt.Time
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
	var idealTempMin, idealTempMax sql.NullFloat64
	var detailsUpdatedAt sql.NullTime
	err := r.pool.QueryRow(ctx, `SELECT id,COALESCE(name,''),COALESCE(description,''),COALESCE(category,''),latitude,longitude,COALESCE(source_url,''),COALESCE(address,''),COALESCE(opening_hours,''),COALESCE(wheelchair,''),COALESCE(website,''),COALESCE(phone,''),COALESCE(fee,''),COALESCE(toilets,''),COALESCE(toilets_wheelchair,''),COALESCE(internet_access,''),COALESCE(wikipedia,''),COALESCE(wikidata,''),details_updated_at,COALESCE(best_climate,''),COALESCE(ideal_weather,''),ideal_temp_min,ideal_temp_max FROM attractions WHERE id=$1`, id).Scan(&place.ID, &place.Name, &place.Description, &place.Category, &place.Latitude, &place.Longitude, &place.SourceURL, &place.Address, &place.OpeningHours, &place.Wheelchair, &place.Website, &place.Phone, &place.Fee, &place.Toilets, &place.ToiletsWheelchair, &place.InternetAccess, &place.Wikipedia, &place.Wikidata, &detailsUpdatedAt, &place.BestClimate, &place.IdealWeather, &idealTempMin, &idealTempMax)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attraction{}, ErrNotFound
	}
	if err != nil {
		return Attraction{}, fmt.Errorf("get attraction: %w", err)
	}
	if idealTempMin.Valid {
		place.IdealTempMin = &idealTempMin.Float64
	}
	if idealTempMax.Valid {
		place.IdealTempMax = &idealTempMax.Float64
	}
	if detailsUpdatedAt.Valid {
		place.DetailsUpdatedAt = &detailsUpdatedAt.Time
	}
	return place, nil
}
