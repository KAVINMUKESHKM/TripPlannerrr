package trips

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"tripplanner/backend/internal/auth"
)

type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
	CheckRequestOrigin(http.ResponseWriter, *http.Request) bool
}

type Handler struct {
	repository    *Repository
	authenticator Authenticator
}

func NewHandler(repository *Repository, authenticator Authenticator) *Handler {
	return &Handler{repository: repository, authenticator: authenticator}
}
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/trips", h.authenticator.RequireAuth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/trips", h.authenticator.RequireAuth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/trips/{id}", h.authenticator.RequireAuth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/trips/{id}", h.authenticator.RequireAuth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/trips/{id}", h.authenticator.RequireAuth(http.HandlerFunc(h.delete)))
	return mux
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	trips, err := h.repository.ListTrips(r.Context(), user.ID)
	if err != nil {
		writeTripError(w, 500, "could not list trips")
		return
	}
	writeTripJSON(w, 200, trips)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if !h.authenticator.CheckRequestOrigin(w, r) {
		return
	}
	var input TripInput
	if !decodeTripJSON(w, r, &input) {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	trip, err := h.repository.CreateTrip(r.Context(), user.ID, input)
	if err != nil {
		writeTripError(w, 400, err.Error())
		return
	}
	writeTripJSON(w, 201, trip)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := tripID(w, r)
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	trip, err := h.repository.GetTrip(r.Context(), user.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeTripError(w, 404, "trip not found")
		return
	}
	if err != nil {
		writeTripError(w, 500, "could not load trip")
		return
	}
	writeTripJSON(w, 200, trip)
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	if !h.authenticator.CheckRequestOrigin(w, r) {
		return
	}
	id, ok := tripID(w, r)
	if !ok {
		return
	}
	var input TripInput
	if !decodeTripJSON(w, r, &input) {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	trip, err := h.repository.UpdateTrip(r.Context(), user.ID, id, input)
	if errors.Is(err, ErrNotFound) {
		writeTripError(w, 404, "trip not found")
		return
	}
	if err != nil {
		writeTripError(w, 400, err.Error())
		return
	}
	writeTripJSON(w, 200, trip)
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if !h.authenticator.CheckRequestOrigin(w, r) {
		return
	}
	id, ok := tripID(w, r)
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	if err := h.repository.DeleteTrip(r.Context(), user.ID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeTripError(w, 404, "trip not found")
		} else {
			writeTripError(w, 500, "could not delete trip")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func tripID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeTripError(w, 400, "trip ID must be a positive integer")
		return 0, false
	}
	return id, true
}
func decodeTripJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeTripError(w, 400, "invalid trip request")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeTripError(w, 400, "request must contain one JSON object")
		return false
	}
	return true
}
func writeTripError(w http.ResponseWriter, status int, message string) {
	writeTripJSON(w, status, map[string]string{"error": message})
}
func writeTripJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
