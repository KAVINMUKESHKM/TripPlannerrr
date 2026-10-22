package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Dependencies struct{ Ready func(context.Context) error }

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/healthz", health)
	mux.HandleFunc("GET /api/v1/readyz", func(w http.ResponseWriter, r *http.Request) {
		if deps.Ready != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := deps.Ready(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
				return
			}
		}
		health(w, r)
	})
	return withTimeout(mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func withTimeout(next http.Handler) http.Handler {
	return timeoutHandler(next, 10*time.Second)
}

func timeoutHandler(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := make(chan struct{})
		buffer := &bufferResponseWriter{header: make(http.Header)}
		go func() { next.ServeHTTP(buffer, r); close(result) }()
		select {
		case <-result:
			for key, values := range buffer.header {
				w.Header()[key] = values
			}
			w.WriteHeader(buffer.status)
			_, _ = w.Write(buffer.body.Bytes())
		case <-time.After(timeout):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request timeout"})
		}
	})
}

type bufferResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *bufferResponseWriter) Header() http.Header { return w.header }
func (w *bufferResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *bufferResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}
