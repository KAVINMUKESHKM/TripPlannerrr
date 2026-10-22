package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Waypoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type RouteSummary struct {
	DistanceMeters  float64 `json:"distanceMeters"`
	DurationSeconds float64 `json:"durationSeconds"`
	Geometry        struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	} `json:"geometry"`
}
type Router struct {
	BaseURL string
	Client  *http.Client
}

func (r Router) Route(ctx context.Context, points []Waypoint) (RouteSummary, error) {
	if len(points) < 2 || len(points) > 25 {
		return RouteSummary{}, errors.New("route requires between 2 and 25 stops")
	}
	coordinates := make([]string, 0, len(points))
	for _, point := range points {
		if point.Latitude < -90 || point.Latitude > 90 || point.Longitude < -180 || point.Longitude > 180 {
			return RouteSummary{}, errors.New("a route coordinate is out of range")
		}
		coordinates = append(coordinates, strconv.FormatFloat(point.Longitude, 'f', 6, 64)+","+strconv.FormatFloat(point.Latitude, 'f', 6, 64))
	}
	base := strings.TrimRight(r.BaseURL, "/")
	if base == "" {
		base = "https://router.project-osrm.org"
	}
	endpoint := base + "/route/v1/driving/" + strings.Join(coordinates, ";") + "?overview=full&geometries=geojson&steps=false"
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RouteSummary{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return RouteSummary{}, errors.New("OSRM routing service is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return RouteSummary{}, fmt.Errorf("OSRM routing service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Geometry struct {
				Type        string       `json:"type"`
				Coordinates [][2]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return RouteSummary{}, errors.New("OSRM response was invalid")
	}
	if body.Code != "Ok" || len(body.Routes) == 0 {
		return RouteSummary{}, errors.New("OSRM could not find a route for these stops")
	}
	route := body.Routes[0]
	return RouteSummary{DistanceMeters: route.Distance, DurationSeconds: route.Duration, Geometry: route.Geometry}, nil
}
