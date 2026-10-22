package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Coordinate struct {
	Latitude  float64
	Longitude float64
}
type Geocoder struct {
	APIKey string
	Client *http.Client
}

func (g Geocoder) Locate(ctx context.Context, query string) (Coordinate, error) {
	if strings.TrimSpace(g.APIKey) == "" {
		return Coordinate{}, errors.New("destination is not in the local database and OPENCAGE_API_KEY is not configured")
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	endpoint := "https://api.opencagedata.com/geocode/v1/json?q=" + url.QueryEscape(query) + "&key=" + url.QueryEscape(g.APIKey) + "&limit=1&no_annotations=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Coordinate{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Coordinate{}, fmt.Errorf("geocoding service is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Coordinate{}, fmt.Errorf("geocoding service returned status %d", resp.StatusCode)
	}
	var body struct {
		Results []struct {
			Geometry struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"geometry"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Coordinate{}, errors.New("geocoding response was invalid")
	}
	if len(body.Results) == 0 {
		return Coordinate{}, errors.New("destination could not be found")
	}
	return Coordinate{Latitude: body.Results[0].Geometry.Lat, Longitude: body.Results[0].Geometry.Lng}, nil
}
