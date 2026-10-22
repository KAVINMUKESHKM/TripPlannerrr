package trips

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("trip not found")

type Trip struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	DestinationID *int64    `json:"destinationId,omitempty"`
	Destination   string    `json:"destination"`
	StartDate     time.Time `json:"startDate"`
	DayCount      int       `json:"dayCount"`
	Days          []Day     `json:"days"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type TripInput struct {
	Destination   string
	DestinationID int64
	StartDate     time.Time
	DayCount      int
	Days          []DayInput
}

type DayInput struct {
	DayNumber     int
	AttractionIDs []int64
}

type Day struct {
	DayNumber int    `json:"dayNumber"`
	Stops     []Stop `json:"stops"`
}

type Stop struct {
	Position     int   `json:"position"`
	AttractionID int64 `json:"attractionId"`
}
