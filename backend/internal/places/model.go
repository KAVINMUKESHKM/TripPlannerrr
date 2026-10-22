package places

import "time"

type Attraction struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Category    string  `json:"category,omitempty"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type SearchDay struct {
	Day         int          `json:"day"`
	Date        string       `json:"date"`
	Weather     string       `json:"weather,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
	Attractions []Attraction `json:"attractions"`
}

type SearchQuery struct {
	Destination string
	Days        int
	Date        time.Time
	Category    string
}
