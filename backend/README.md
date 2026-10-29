# Go backend

The Go API serves authentication, attraction search, nearby restaurant results, routing, and saved-trip operations. It uses PostgreSQL and loads `backend/.env` when run from this directory.

## Requirements

- Go 1.27.3 (see `go.mod`)
- PostgreSQL
- Optional provider API keys listed below

## Configure

Copy `.env.example` to `.env` only if `.env` does not already exist. Set `DATABASE_URL` to the PostgreSQL runtime connection. Keep secrets out of source control.

| Variable | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `MIGRATOR_DATABASE_URL` | No | Separate schema-owner connection used only by the migration command; defaults to `DATABASE_URL` |
| `IMPORTER_DATABASE_URL` | No | Optional data-loader connection used only by `cmd/importosm`; defaults to `DATABASE_URL` |
| `PORT` | No | API port; defaults to `8080` |
| `FRONTEND_URL` | Recommended | Allowed browser origin and OAuth return target; defaults to local frontend when configured |
| `OPENCAGE_API_KEY` | No | Geocode destination names missing from `destinations` |
| `OPENWEATHER_API_KEY` (`OPEN_WEATHER_KEY` also accepted) | No | Add daily weather summaries and match saved ideal-weather preferences |
| `SERP_API_KEY` (`SERP_KEY` also accepted) | No | Find nearby restaurants from attraction cards |
| `GOOGLE_CLIENT_ID` | No | Enables Google OAuth when paired with the secret |
| `GOOGLE_CLIENT_SECRET` | No | Enables Google OAuth when paired with the client ID |
| `GOOGLE_REDIRECT_URL` | No | OAuth callback; defaults to `${FRONTEND_URL}/api/v1/auth/google/callback` |
| `COOKIE_SECURE` | No | Set `true` for HTTPS deployments; use `false` locally |
| `APP_ENV` | No | Set `production` in deployment; requires secure cookies and a TLS-only PostgreSQL URL |
| `TRUSTED_PROXIES` | No | Comma-separated trusted proxy CIDRs; only these peers may supply client IPs through `X-Forwarded-For` |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | No | Configure all five to enable email password reset |
| `OSRM_BASE_URL` | No | OSRM endpoint; defaults to the public OSRM router |
| `OVERPASS_URL` | No | OpenStreetMap Overpass endpoint used by `cmd/enrichosm`; defaults to the Private.coffee public instance |

## Run

From `backend/`:

```bash
go run ./cmd/migrate up
go run ./cmd/enrichosm                 # fetch current visitor details from OSM/Overpass and Wikidata
go run ./cmd/enrichosm --wikidata-only # refresh descriptions and websites without Overpass
go run ./cmd/importosm                 # optional: import bundled Tamil Nadu data
go run ./cmd/importosm data/places.json # optional: import another compatible dataset
go run ./cmd/api
```

The migration command applies unapplied SQL migrations in filename order. The API does not run migrations automatically. `enrichosm` fetches one batch of current OpenStreetMap tags for the bundled records and updates the JSON dataset; `importosm` then upserts those details into PostgreSQL. Both commands can be rerun. Detail coverage depends on what OpenStreetMap contributors have mapped; empty fields mean no value was available, not that a venue has no hours or accessibility features.

At startup the API logs database readiness, its listen address, and (when OAuth is enabled) the exact Google callback URI. Each request is logged with its method, path, status, and duration; query strings and credentials are omitted. Check the callback log against the authorized redirect URI in Google Cloud if Google reports `redirect_uri_mismatch`.

## API routes

All routes use the `/api/v1` prefix.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness check |
| `GET` | `/readyz` | Database readiness check |
| `POST` | `/auth/register` | Create an account and session |
| `POST` | `/auth/login` | Start an email/password session |
| `POST` | `/auth/logout` | End the current session |
| `POST` | `/auth/logout-all` | Revoke all sessions on every device |
| `POST` | `/auth/password/forgot` | Send a one-time password-reset link (SMTP required) |
| `POST` | `/auth/password/reset` | Change password using a valid reset token |
| `GET` | `/auth/me` | Return the current signed-in user |
| `GET` | `/auth/google` | Start Google OAuth |
| `GET` | `/auth/google/callback` | Complete Google OAuth |
| `GET` | `/attractions?destination=…&days=…&date=YYYY-MM-DD&category=…` | Search and group attractions by itinerary day |
| `GET` | `/attractions/{id}` | Get one attraction |
| `GET` | `/attractions/{id}/restaurants` | Get nearby restaurants (SerpAPI key required) |
| `POST` | `/routes` | Get an OSRM route for ordered waypoints |
| `GET`, `POST` | `/trips` | List or create the signed-in user's trips |
| `GET`, `PUT`, `DELETE` | `/trips/{id}` | Read, update, or delete an owned trip |

Sessions use HttpOnly cookies. Trip routes require a signed-in user. For local Google sign-in, register `http://localhost:3000/api/v1/auth/google/callback` as an authorized redirect URI in Google Cloud. The frontend proxies this callback to the API. The API logs the exact callback URI at startup; if you set `GOOGLE_REDIRECT_URL`, register that exact value instead. Add your production HTTPS callback separately for deployment.

For production, serve the frontend over HTTPS, set `APP_ENV=production` and `COOKIE_SECURE=true`, use PostgreSQL connection strings with TLS enabled (for example `sslmode=require`), and give the API database user only the permissions it needs. Rate windows are stored in PostgreSQL and shared by all instances. If the API is behind a proxy, set `TRUSTED_PROXIES` to that proxy's exact CIDR so limits use the real client IP; the API ignores forwarded-IP headers from untrusted peers.

Use a schema-owner role for migrations, a write-limited importer role for `cmd/importosm`, and a separate API role for runtime queries. For a database named `tripplanner`, grant the runtime role only the access it needs after migrations have run:

```sql
GRANT CONNECT ON DATABASE tripplanner TO tripplanner_app;
GRANT USAGE ON SCHEMA public TO tripplanner_app;
GRANT SELECT, INSERT, UPDATE ON users TO tripplanner_app;
GRANT SELECT, INSERT, DELETE ON sessions, oauth_states TO tripplanner_app;
GRANT SELECT, INSERT, DELETE ON password_reset_tokens TO tripplanner_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON api_rate_limits TO tripplanner_app;
GRANT SELECT ON destinations, attractions TO tripplanner_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON trips, trip_days, trip_stops TO tripplanner_app;
GRANT USAGE, SELECT ON SEQUENCE users_id_seq, trips_id_seq TO tripplanner_app;

GRANT CONNECT ON DATABASE tripplanner TO tripplanner_importer;
GRANT USAGE ON SCHEMA public TO tripplanner_importer;
GRANT SELECT, INSERT, UPDATE ON attractions TO tripplanner_importer;
GRANT USAGE, SELECT ON SEQUENCE attractions_id_seq TO tripplanner_importer;
```

Set `DATABASE_URL` to the runtime role, `MIGRATOR_DATABASE_URL` to the schema-owner role, and `IMPORTER_DATABASE_URL` to the importer role. Reapply the table grants when new migrations add runtime tables. Configure SMTP to enable password-reset emails; without it, reset requests return a service-unavailable response and no reset tokens are issued.

## Interactive API documentation

Start the API, then open [http://localhost:8080/docs](http://localhost:8080/docs) for Swagger UI. The OpenAPI 3.0 specification is served at [http://localhost:8080/openapi.yaml](http://localhost:8080/openapi.yaml). Swagger UI loads its pinned 5.33.3 interface assets from unpkg, so the interactive page needs an internet connection; the YAML specification is served directly by the Go API.

The frontend reads `NEXT_PUBLIC_OPENFREEMAP_STYLE_URL` for its MapLibre style; this is configured in `frontend/.env.local`, not in the backend.

## Development checks

Tests are grouped under `tests/unit` and `tests/integration`; shared PostGIS fixtures live under `tests/testutil`. A couple of white-box tests stay beside their packages because they cover unexported behavior.

```bash
go test ./tests/unit/...
go test ./tests/integration/... # requires Docker for PostGIS
go test ./...
go build ./...
```

Place details show hours, address, accessibility, fees, facilities, and website/reference links where OpenStreetMap has those tags. Verify time-sensitive details with the venue before visiting. The attraction dataset is OpenStreetMap data. Read [`data/README.md`](data/README.md) for attribution and license notes.
