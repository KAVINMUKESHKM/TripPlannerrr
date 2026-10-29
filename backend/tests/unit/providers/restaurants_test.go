package providers_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"tripplanner/backend/internal/providers"
)

func TestRestaurantsFiltersAndSortsNearbyPlaces(t *testing.T) {
	provider := providers.Restaurants{APIKey: "test-key", Endpoint: "https://serp.test/search.json", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.Query().Get("engine"); got != "google_maps" {
			t.Errorf("engine = %q, want google_maps", got)
		}
		if got := r.URL.Query().Get("radius"); got != "2000" {
			t.Errorf("radius = %q, want 2000", got)
		}
		if got := r.URL.Query().Get("api_key"); got != "test-key" {
			t.Errorf("api_key = %q, want test-key", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"local_results":[
			{"title":"Far","address":"Far Road","rating":4.8,"phone":"123","gps_coordinates":{"latitude":13.1,"longitude":80.3}},
			{"title":"Low rated","rating":2.9,"gps_coordinates":{"latitude":13.001,"longitude":80.001}},
			{"title":"Near","address":"Near Road","rating":4.1,"gps_coordinates":{"latitude":13.001,"longitude":80.001}},
			{"title":"Second near","rating":4.0,"gps_coordinates":{"latitude":13.002,"longitude":80.002}},
			{"title":"Third near","rating":4.0,"gps_coordinates":{"latitude":13.003,"longitude":80.003}},
			{"title":"No coordinates","rating":5.0}
		]}`)), Request: r}, nil
	})}}

	got, err := provider.Nearby(context.Background(), providers.Coordinate{Latitude: 13, Longitude: 80})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d restaurants, want nearest 2", len(got))
	}
	if got[0].Name != "Near" || got[1].Name != "Second near" {
		t.Fatalf("restaurant order = %q, %q; want Near, Second near", got[0].Name, got[1].Name)
	}
	if got[0].DistanceKM <= 0 || got[0].DistanceKM >= got[1].DistanceKM {
		t.Fatalf("distances not ordered: %#v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestRestaurantsRequiresAPIKey(t *testing.T) {
	_, err := (providers.Restaurants{}).Nearby(context.Background(), providers.Coordinate{Latitude: 13, Longitude: 80})
	if err == nil {
		t.Fatal("Nearby() without a key succeeded")
	}
}
