package places_test

import (
	"context"
	"testing"
	"time"

	"tripplanner/backend/internal/places"
	"tripplanner/backend/internal/providers"
)

type testStore struct{ places []places.Attraction }

func (s testStore) FindDestination(context.Context, string) (providers.Coordinate, error) {
	return providers.Coordinate{Latitude: 13, Longitude: 80}, nil
}
func (s testStore) Search(context.Context, providers.Coordinate, float64, string) ([]places.Attraction, error) {
	return s.places, nil
}

type testGeocoder struct{}

func (testGeocoder) Locate(context.Context, string) (providers.Coordinate, error) {
	return providers.Coordinate{}, nil
}

type testWeather struct{ forecasts []providers.Forecast }

func (w testWeather) Daily(context.Context, providers.Coordinate, int) ([]providers.Forecast, error) {
	return w.forecasts, nil
}

func TestSearchAppliesLegacyWeatherPreferencesAndKeepsUnspecifiedPlaces(t *testing.T) {
	service := places.Service{
		Store: testStore{places: []places.Attraction{
			{ID: 1, Name: "Any weather", IdealWeather: ""},
			{ID: 2, Name: "Rain garden", IdealWeather: "rainy"},
			{ID: 3, Name: "Cloud garden", IdealWeather: "cloudy"},
			{ID: 4, Name: "Waterfall", Category: "waterfall"},
		}},
		Geocoder: testGeocoder{},
		Weather:  testWeather{forecasts: []providers.Forecast{{Date: "2026-10-29", Condition: "rain", Hour: 12}}},
	}
	got, err := service.Search(context.Background(), places.SearchQuery{Destination: "Chennai", Days: 1, Date: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d days, want one", len(got))
	}
	if len(got[0].Attractions) != 3 {
		t.Fatalf("got %d suggestions, want 3", len(got[0].Attractions))
	}
	for _, place := range got[0].Attractions {
		if place.ID == 3 {
			t.Fatal("cloud-preferred attraction was suggested in rain")
		}
	}
}

func TestSearchRestrictsWaterfallsToForecastDaytime(t *testing.T) {
	service := places.Service{
		Store:    testStore{places: []places.Attraction{{ID: 9, Name: "Waterfall", Category: "Waterfall"}}},
		Geocoder: testGeocoder{},
		Weather:  testWeather{forecasts: []providers.Forecast{{Date: "2026-10-29", Condition: "clear", Hour: 18}}},
	}
	got, err := service.Search(context.Background(), places.SearchQuery{Destination: "Chennai", Days: 1, Date: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Attractions) != 0 {
		t.Fatalf("got %d waterfall suggestions outside visiting hours", len(got[0].Attractions))
	}
}
