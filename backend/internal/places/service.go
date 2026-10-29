package places

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tripplanner/backend/internal/providers"
)

type Store interface {
	FindDestination(context.Context, string) (providers.Coordinate, error)
	Search(context.Context, providers.Coordinate, float64, string) ([]Attraction, error)
}
type Geocoder interface {
	Locate(context.Context, string) (providers.Coordinate, error)
}
type WeatherProvider interface {
	Daily(context.Context, providers.Coordinate, int) ([]providers.Forecast, error)
}
type Service struct {
	Store    Store
	Geocoder Geocoder
	Weather  WeatherProvider
}

func (s Service) Search(ctx context.Context, query SearchQuery) ([]SearchDay, error) {
	query.Destination = strings.TrimSpace(query.Destination)
	if query.Destination == "" {
		return nil, errors.New("destination is required")
	}
	if query.Days < 1 || query.Days > 14 {
		return nil, errors.New("days must be between 1 and 14")
	}
	if query.Date.IsZero() {
		return nil, errors.New("date is required")
	}
	point, err := s.Store.FindDestination(ctx, query.Destination)
	if errors.Is(err, ErrNotFound) {
		point, err = s.Geocoder.Locate(ctx, query.Destination)
	}
	if err != nil {
		return nil, err
	}
	radius := 12.0
	if query.Days > 1 {
		radius = 30
	}
	places, err := s.Store.Search(ctx, point, radius, query.Category)
	if err != nil {
		return nil, err
	}
	forecasts, _ := s.Weather.Daily(ctx, point, query.Days) // Weather is optional; place discovery remains available during provider outages.
	if len(forecasts) > 0 {
		places = filterWeatherSuggestions(places, forecasts)
	}
	days := make([]SearchDay, query.Days)
	for i := range days {
		date := query.Date.AddDate(0, 0, i)
		days[i] = SearchDay{Day: i + 1, Date: date.Format("2006-01-02"), Attractions: make([]Attraction, 0)}
	}
	if len(places) > 0 {
		perDay := (len(places) + query.Days - 1) / query.Days
		for i, place := range places {
			dayIndex := i / perDay
			if dayIndex >= len(days) {
				dayIndex = len(days) - 1
			}
			days[dayIndex].Attractions = append(days[dayIndex].Attractions, place)
		}
	}
	for i, forecast := range forecasts {
		if i >= len(days) {
			break
		}
		if forecast.Date == days[i].Date {
			days[i].Weather = forecast.Condition
			temperature := forecast.Temperature
			days[i].Temperature = &temperature
		}
	}
	return days, nil
}

func filterWeatherSuggestions(places []Attraction, forecasts []providers.Forecast) []Attraction {
	eligible := make(map[int64]bool, len(places))
	for _, forecast := range forecasts {
		condition := strings.ToLower(forecast.Condition)
		if condition == "" {
			continue
		}
		for _, place := range places {
			if strings.EqualFold(place.Category, "waterfall") && (forecast.Hour < 10 || forecast.Hour > 16) {
				continue
			}
			ideal := strings.ToLower(strings.TrimSpace(place.IdealWeather))
			if ideal != "" && condition == "clouds" && !strings.Contains(ideal, "cloud") {
				continue
			}
			if ideal != "" && condition == "rain" && !strings.Contains(ideal, "rain") {
				continue
			}
			eligible[place.ID] = true
		}
	}
	filtered := make([]Attraction, 0, len(eligible))
	for _, place := range places {
		if eligible[place.ID] {
			filtered = append(filtered, place)
		}
	}
	return filtered
}

func ParseSearchQuery(destination string, days, date, category string) (SearchQuery, error) {
	if strings.TrimSpace(destination) == "" {
		return SearchQuery{}, errors.New("destination is required")
	}
	count := 0
	if _, err := fmt.Sscanf(days, "%d", &count); err != nil || count < 1 || count > 14 {
		return SearchQuery{}, errors.New("days must be between 1 and 14")
	}
	start, err := time.Parse("2006-01-02", date)
	if err != nil {
		return SearchQuery{}, errors.New("date must use YYYY-MM-DD")
	}
	return SearchQuery{Destination: destination, Days: count, Date: start, Category: category}, nil
}
