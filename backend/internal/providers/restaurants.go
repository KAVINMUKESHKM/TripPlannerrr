package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Restaurant struct {
	Name       string  `json:"name"`
	Address    string  `json:"address,omitempty"`
	Rating     float64 `json:"rating"`
	Phone      string  `json:"phone,omitempty"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	DistanceKM float64 `json:"distanceKm"`
}

type Restaurants struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
}

func (p Restaurants) Nearby(ctx context.Context, point Coordinate) ([]Restaurant, error) {
	if strings.TrimSpace(p.APIKey) == "" {
		return nil, errors.New("nearby restaurant search is not configured (SERP_API_KEY is missing)")
	}
	endpoint := strings.TrimRight(p.Endpoint, "/")
	if endpoint == "" {
		endpoint = "https://serpapi.com/search.json"
	}
	query := url.Values{
		"engine":  {"google_maps"},
		"q":       {"restaurant"},
		"radius":  {"2000"},
		"ll":      {fmt.Sprintf("@%s,%s,15.1z", strconv.FormatFloat(point.Latitude, 'f', 6, 64), strconv.FormatFloat(point.Longitude, 'f', 6, 64))},
		"type":    {"search"},
		"api_key": {p.APIKey},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, errors.New("could not create restaurant search request")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("restaurant search service is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("restaurant search service returned status %d", resp.StatusCode)
	}
	var body struct {
		Results []struct {
			Title       string  `json:"title"`
			Address     string  `json:"address"`
			Rating      float64 `json:"rating"`
			Phone       string  `json:"phone"`
			Coordinates struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"gps_coordinates"`
		} `json:"local_results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, errors.New("restaurant search response was invalid")
	}
	results := make([]Restaurant, 0, len(body.Results))
	for _, item := range body.Results {
		if strings.TrimSpace(item.Title) == "" || item.Rating < 3 || item.Coordinates.Latitude < -90 || item.Coordinates.Latitude > 90 || item.Coordinates.Longitude < -180 || item.Coordinates.Longitude > 180 || (item.Coordinates.Latitude == 0 && item.Coordinates.Longitude == 0) {
			continue
		}
		results = append(results, Restaurant{
			Name: item.Title, Address: item.Address, Rating: item.Rating, Phone: item.Phone,
			Latitude: item.Coordinates.Latitude, Longitude: item.Coordinates.Longitude,
			DistanceKM: haversineKM(point, Coordinate{Latitude: item.Coordinates.Latitude, Longitude: item.Coordinates.Longitude}),
		})
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].DistanceKM < results[j].DistanceKM })
	if len(results) > 2 {
		results = results[:2]
	}
	return results, nil
}

func haversineKM(a, b Coordinate) float64 {
	const earthRadiusKM = 6371.0088
	lat1, lat2 := a.Latitude*math.Pi/180, b.Latitude*math.Pi/180
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKM * math.Asin(math.Sqrt(h))
}
