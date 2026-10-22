package places

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tripplanner/backend/internal/providers"
)

type RouteProvider interface {
	Route(context.Context, []providers.Waypoint) (providers.RouteSummary, error)
}
type Handler struct {
	SearchService Service
	Store         *Repository
	Router        RouteProvider
}

func NewHandler(repo *Repository, geocoder Geocoder, weather WeatherProvider, router RouteProvider) *Handler {
	return &Handler{SearchService: Service{Store: repo, Geocoder: geocoder, Weather: weather}, Store: repo, Router: router}
}
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/attractions", h.search)
	mux.HandleFunc("GET /api/v1/attractions/{id}", h.get)
	mux.HandleFunc("POST /api/v1/routes", h.route)
	return mux
}
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q, err := ParseSearchQuery(r.URL.Query().Get("destination"), r.URL.Query().Get("days"), r.URL.Query().Get("date"), r.URL.Query().Get("category"))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	result, err := h.SearchService.Search(r.Context(), q)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "not configured") {
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, 200, result)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "attraction ID must be a positive integer")
		return
	}
	place, err := h.Store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, 404, "attraction not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load attraction")
		return
	}
	writeJSON(w, 200, place)
}
func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Waypoints []providers.Waypoint `json:"waypoints"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, 400, "invalid route request")
		return
	}
	result, err := h.Router.Route(r.Context(), input.Waypoints)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, result)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
