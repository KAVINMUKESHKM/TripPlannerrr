package httpapi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// RequestLimiter stores rate windows in a shared backend so all API instances
// enforce the same limits.
type RequestLimiter interface {
	Allow(context.Context, string, *http.Request, int, time.Duration) (bool, time.Duration, error)
}

type RateLimitQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresRateLimiter struct {
	db             RateLimitQueryer
	trustedProxies []netip.Prefix
}

func NewPostgresRateLimiter(db RateLimitQueryer, trustedProxyCIDRs string) (*PostgresRateLimiter, error) {
	limiter := &PostgresRateLimiter{db: db}
	for _, value := range strings.Split(trustedProxyCIDRs, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q", value)
		}
		limiter.trustedProxies = append(limiter.trustedProxies, prefix.Masked())
	}
	return limiter, nil
}

func (l *PostgresRateLimiter) Allow(ctx context.Context, group string, request *http.Request, limit int, window time.Duration) (bool, time.Duration, error) {
	keyHash := sha256.Sum256([]byte(group + ":" + clientIP(request, l.trustedProxies)))
	var allowed bool
	var retrySeconds int
	err := l.db.QueryRow(ctx, `INSERT INTO api_rate_limits (key_hash, window_started, request_count)
		VALUES ($1, clock_timestamp(), 1)
		ON CONFLICT (key_hash) DO UPDATE SET
		window_started = CASE
			WHEN api_rate_limits.window_started <= statement_timestamp() - make_interval(secs => $3::double precision)
			THEN statement_timestamp() ELSE api_rate_limits.window_started END,
		request_count = CASE
			WHEN api_rate_limits.window_started <= statement_timestamp() - make_interval(secs => $3::double precision)
			THEN 1 ELSE LEAST(api_rate_limits.request_count + 1, $2) END
		RETURNING request_count <= $2,
			GREATEST(1, CEIL(EXTRACT(EPOCH FROM (window_started + make_interval(secs => $3::double precision) - statement_timestamp()))))::integer`,
		keyHash[:], limit, window.Seconds()).Scan(&allowed, &retrySeconds)
	if err != nil {
		return false, 0, err
	}
	return allowed, time.Duration(retrySeconds) * time.Second, nil
}

func rateLimitMiddleware(limiter RequestLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, window, group := 0, time.Minute, ""
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/auth/"):
			limit, group = 10, "auth"
		case r.URL.Path == "/api/v1/attractions" && r.Method == http.MethodGet:
			limit, group = 30, "places"
		case strings.HasSuffix(r.URL.Path, "/restaurants"):
			limit, group = 10, "restaurants"
		case r.URL.Path == "/api/v1/routes" && r.Method == http.MethodPost:
			limit, group = 30, "routes"
		}
		if limit != 0 {
			allowed, retryAfter, err := limiter.Allow(r.Context(), group, r, limit, window)
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request limits are temporarily unavailable"})
				return
			}
			if !allowed {
				if retryAfter < time.Second {
					retryAfter = time.Second
				}
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Round(time.Second)/time.Second)))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests; try again shortly"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request, trusted []netip.Prefix) string {
	peer := remoteIP(r.RemoteAddr)
	if peer == "" {
		return r.RemoteAddr
	}
	peerAddr, err := netip.ParseAddr(peer)
	if err != nil || !isTrustedProxy(peerAddr, trusted) {
		return peer
	}
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		address, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil {
			return peer
		}
		if !isTrustedProxy(address, trusted) {
			return address.String()
		}
	}
	return peer
}

func remoteIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func isTrustedProxy(address netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
