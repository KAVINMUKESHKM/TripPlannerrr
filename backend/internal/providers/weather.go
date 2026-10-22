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

type Forecast struct {
	Date        string  `json:"date"`
	Condition   string  `json:"condition"`
	Temperature float64 `json:"temperature"`
}
type Weather struct {
	APIKey string
	Client *http.Client
}

func (w Weather) Daily(ctx context.Context, point Coordinate, days int) ([]Forecast, error) {
	if strings.TrimSpace(w.APIKey) == "" {
		return nil, nil
	}
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	query := url.Values{"lat": {fmt.Sprint(point.Latitude)}, "lon": {fmt.Sprint(point.Longitude)}, "appid": {w.APIKey}, "units": {"metric"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openweathermap.org/data/2.5/forecast?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("weather service is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather service returned status %d", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			DateTime string `json:"dt_txt"`
			Main     struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				Main string `json:"main"`
			} `json:"weather"`
		} `json:"list"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, errors.New("weather response was invalid")
	}
	byDate := make(map[string]Forecast)
	for _, item := range body.Items {
		if len(item.Weather) == 0 {
			continue
		}
		date := item.DateTime
		if len(date) > 10 {
			date = date[:10]
		}
		if _, exists := byDate[date]; !exists || strings.Contains(item.DateTime, "12:00:00") {
			byDate[date] = Forecast{Date: date, Condition: strings.ToLower(item.Weather[0].Main), Temperature: item.Main.Temp}
		}
	}
	forecasts := make([]Forecast, 0, len(byDate))
	for _, forecast := range byDate {
		forecasts = append(forecasts, forecast)
	}
	// Sort the provider's YYYY-MM-DD keys without requiring locale-aware date parsing.
	for i := 1; i < len(forecasts); i++ {
		for j := i; j > 0 && forecasts[j].Date < forecasts[j-1].Date; j-- {
			forecasts[j], forecasts[j-1] = forecasts[j-1], forecasts[j]
		}
	}
	if len(forecasts) > days {
		forecasts = forecasts[:days]
	}
	return forecasts, nil
}
