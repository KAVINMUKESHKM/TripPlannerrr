package httpapi

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

//go:embed openapi.yaml
var openAPISpec []byte

type Dependencies struct {
	Ready   func(context.Context) error
	Limiter RequestLimiter
	Auth    http.Handler
	Places  http.Handler
	Trips   http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /docs", swaggerDocs)
	mux.HandleFunc("GET /swagger/", swaggerDocs)
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("GET /openapi.yaml", serveOpenAPISpec)
	mux.HandleFunc("GET /api/v1/healthz", health)
	if deps.Auth != nil {
		mux.Handle("/api/v1/auth/", deps.Auth)
	}
	if deps.Places != nil {
		mux.Handle("/api/v1/attractions", deps.Places)
		mux.Handle("/api/v1/attractions/", deps.Places)
		mux.Handle("/api/v1/routes", deps.Places)
	}
	if deps.Trips != nil {
		mux.Handle("/api/v1/trips", deps.Trips)
		mux.Handle("/api/v1/trips/", deps.Trips)
	}
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
	if deps.Limiter == nil {
		return withTimeout(securityHeaders(mux))
	}
	return withTimeout(securityHeaders(rateLimitMiddleware(deps.Limiter, mux)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(openAPISpec)
}

func swaggerDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Trip Planner API — Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.33.3/swagger-ui.css" />
    <style>body{margin:0;background:#eae0d5}.swagger-ui .topbar{background:#22333b}.swagger-ui .topbar-wrapper .link span{visibility:hidden}.swagger-ui .topbar-wrapper .link:after{content:"Trip Planner API";color:#eae0d5;font-weight:600;visibility:visible}</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.33.3/swagger-ui-bundle.js"></script>
    <script>window.onload=()=>window.SwaggerUIBundle({url:"/openapi.yaml",dom_id:"#swagger-ui",deepLinking:true,persistAuthorization:false,displayRequestDuration:true,filter:true,tryItOutEnabled:true,withCredentials:true});</script>
  </body>
</html>`))
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
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
