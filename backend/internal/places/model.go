package places

import "time"

type Attraction struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Description       string     `json:"description,omitempty"`
	Category          string     `json:"category,omitempty"`
	Latitude          float64    `json:"latitude"`
	Longitude         float64    `json:"longitude"`
	BestClimate       string     `json:"best_climate,omitempty"`
	IdealWeather      string     `json:"ideal_weather,omitempty"`
	IdealTempMin      *float64   `json:"ideal_temp_min,omitempty"`
	IdealTempMax      *float64   `json:"ideal_temp_max,omitempty"`
	SourceURL         string     `json:"source_url,omitempty"`
	Address           string     `json:"address,omitempty"`
	OpeningHours      string     `json:"opening_hours,omitempty"`
	Wheelchair        string     `json:"wheelchair,omitempty"`
	Website           string     `json:"website,omitempty"`
	Phone             string     `json:"phone,omitempty"`
	Fee               string     `json:"fee,omitempty"`
	Toilets           string     `json:"toilets,omitempty"`
	ToiletsWheelchair string     `json:"toilets_wheelchair,omitempty"`
	InternetAccess    string     `json:"internet_access,omitempty"`
	Wikipedia         string     `json:"wikipedia,omitempty"`
	Wikidata          string     `json:"wikidata,omitempty"`
	DetailsUpdatedAt  *time.Time `json:"details_updated_at,omitempty"`
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
