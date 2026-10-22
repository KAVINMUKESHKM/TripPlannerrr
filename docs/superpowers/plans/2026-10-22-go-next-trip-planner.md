# Trip Planner Go and Next.js Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Express/Vite application with a Go 1.27.3 API and latest stable Next.js app, retaining trip discovery and adding Google OAuth plus saved editable itineraries.

**Architecture:** A Go `net/http` JSON API uses PostgreSQL through `pgx/v5`, with explicit repositories, services, migrations, and opaque server-side sessions. The Next.js App Router frontend calls the API through a typed client, and loads MapLibre only on itinerary/map views, using OpenFreeMap styles/tiles and OSRM routes. Existing attractions/destinations stay in PostgreSQL; itinerary data is normalized and owned by the authenticated user.

**Tech Stack:** Go 1.27.3, `net/http`, `pgx/v5`, `golang.org/x/crypto/bcrypt`, `golang.org/x/oauth2`, Next.js 16.4.2 (or latest stable at install time), TypeScript, PostgreSQL/PostGIS, MapLibre GL JS, OpenFreeMap, OSRM, OpenCage, OpenWeather.

**Spec:** `docs/superpowers/specs/2026-10-22-go-next-trip-planner-design.md`

## Global Constraints

- Use Go `1.27.3`.
- Install the latest stable Next.js version available at implementation time; avoid prereleases.
- Preserve PostgreSQL/PostGIS attractions, destinations, and existing password user identities and hashes.
- Do not run database DDL during API startup; apply versioned, repeatable migrations.
- Keep provider secrets server-side; map style/tile and OSRM endpoints are configurable.
- OAuth must validate state and verified identity; sessions use Secure, HttpOnly cookies in production.
- Every itinerary read/write is scoped to the authenticated user in SQL.
- No Leaflet dependencies remain; render maps with MapLibre using OpenFreeMap styles/tiles and route with OSRM.
- Provider outages or missing keys must not prevent health checks or non-provider pages from loading.

## Review Focus

- Legacy PostgreSQL schemas may differ from the incomplete startup DDL; migration must work against the actual deployed tables without dropping or renumbering existing users/attractions. Test migration against a legacy schema fixture.
- Concurrent OAuth callbacks or login requests must not reuse state or create duplicate accounts. Test invalid, expired, and replayed state and duplicate verified email handling.
- A user must not read or mutate another user's itinerary by changing an ID. Test every itinerary operation with a foreign owner.
- Missing/invalid map, weather, or geocoding credentials must return actionable errors without leaking secrets. Test each provider's unconfigured path.
- Empty, one-stop, and many-stop itineraries must remain usable on mobile, with route/map fallbacks. Test empty/one/multiple stop states and a narrow viewport.

---

### Task 1: Establish Go API and Next.js application shells

**Files:**
- Create: `backend/go.mod`, `backend/cmd/api/main.go`, `backend/internal/config/config.go`, `backend/internal/httpapi/router.go`, `backend/.env.example`
- Create: `frontend/package.json`, `frontend/tsconfig.json`, `frontend/next.config.ts`, `frontend/src/app/layout.tsx`, `frontend/src/app/page.tsx`, `frontend/.env.example`
- Modify: root `package.json`, `.gitignore`
- Test: `backend/internal/config/config_test.go`, `backend/internal/httpapi/router_test.go`

**Interfaces:**
- Produces: `config.Load() (Config, error)`; `httpapi.NewRouter(Dependencies) http.Handler`; `GET /api/v1/healthz` and `GET /api/v1/readyz`.

- [ ] **Step 1: Write failing config and health/readiness tests** for required database URL, optional provider keys, live health response, and readiness response without secret fields.
- [ ] **Step 2: Run targeted Go tests** with `cd backend && go test ./internal/config ./internal/httpapi`; confirm expected missing package/behavior failures.
- [ ] **Step 3: Implement configuration validation and the `net/http` router**, with bounded timeouts and structured JSON errors.
- [ ] **Step 4: Scaffold the Next.js App Router application** with a responsive layout, global font/color tokens, and root scripts for frontend dev/build and backend run/test.
- [ ] **Step 5: Re-run targeted Go tests and build both empty app shells** with `cd backend && go test ./...` and `cd frontend && npm run build`.

### Task 2: Preserve existing data and add normalized itinerary schema

**Files:**
- Create: `backend/migrations/0001_users_and_trips.up.sql`, `backend/migrations/0001_users_and_trips.down.sql`, `backend/internal/db/migrate.go`, `backend/internal/db/migrate_test.go`, `backend/cmd/migrate/main.go`
- Create: `backend/internal/trips/model.go`, `backend/internal/trips/repository.go`, `backend/internal/trips/repository_test.go`
- Modify: `backend/internal/config/config.go`

**Interfaces:**
- Produces: migration command `go run ./cmd/migrate up`; repository methods `CreateTrip(ctx, userID, TripInput)`, `ListTrips(ctx, userID)`, `GetTrip(ctx, userID, tripID)`, `UpdateTrip(ctx, userID, tripID, TripInput)`, `DeleteTrip(ctx, userID, tripID)`.
- `TripInput` contains destination, start date, day count, and ordered day/attraction IDs; API and database IDs use stable existing attraction IDs.

- [x] **Step 1: Add PostgreSQL integration tests** using a temporary test database/schema fixture representing existing `users`, `destinations`, and `attractions`, including PostGIS coordinates and a password hash; assert old rows/IDs remain unchanged and trip create/list/update/delete round-trips.
- [x] **Step 2: Run the new integration tests against the legacy fixture** and confirm they fail before migration/repository implementation.
- [x] **Step 3: Implement additive migrations** for OAuth subject, sessions, trips, trip days, and ordered stops; do not redefine/drop legacy tables or run DDL on API startup.
- [x] **Step 4: Implement `pgx/v5` migration runner and parameterized user-scoped trip repository** with transaction-safe day/stop replacement.
- [x] **Step 5: Run migration and repository tests**, including a second migration run and ownership-filtered lookups.

### Task 3: Implement secure password auth and opaque sessions

**Files:**
- Create: `backend/internal/auth/password.go`, `backend/internal/auth/session.go`, `backend/internal/auth/handler.go`, `backend/internal/auth/auth_test.go`
- Modify: `backend/internal/httpapi/router.go`, `backend/internal/config/config.go`
- Test: `backend/internal/auth/auth_integration_test.go`

**Interfaces:**
- Produces: `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`; middleware injects authenticated `User` into request context.
- Cookie contains a random opaque session secret; database stores only its hash with expiry and user ID.

- [x] **Step 1: Write failing handler tests** for registration, duplicate email, wrong password, success cookies, session expiry, logout invalidation, and no password/hash in JSON.
- [x] **Step 2: Run targeted auth tests** and verify failure before implementation.
- [x] **Step 3: Implement password hashing and auth repository queries** while retaining existing user IDs/password hashes.
- [x] **Step 4: Implement opaque database-backed sessions**, cookie settings, origin/CSRF checks on mutations, and auth middleware.
- [x] **Step 5: Run auth tests**, including a login against a pre-migration legacy user record.

### Task 4: Add Google OAuth sign-in

**Files:**
- Create: `backend/internal/auth/google_oauth.go`, `backend/internal/auth/google_oauth_test.go`
- Modify: `backend/internal/config/config.go`, `backend/internal/httpapi/router.go`, `backend/internal/auth/handler.go`

**Interfaces:**
- Produces: `GET /api/v1/auth/google` and `GET /api/v1/auth/google/callback`; Google identity is accepted only after ID token signature, audience, issuer, expiry, and verified email validation.
- Builder call fixed here: if a verified Google email matches an existing password account, attach the Google subject to that account without changing its ID or password; reject unverified/missing email and duplicate subject conflicts.

- [ ] **Step 1: Write OAuth tests** for state mismatch/replay/expiry, invalid token claims, verified account linking, new account creation, and provider cancellation.
- [ ] **Step 2: Run the OAuth tests** and confirm they fail before implementation.
- [ ] **Step 3: Implement authorization code flow** using `golang.org/x/oauth2`, cryptographically random single-use state stored server-side, and verified Google identity validation.
- [ ] **Step 4: Issue the same opaque session type as password login** and redirect to a configured frontend callback destination.
- [ ] **Step 5: Run OAuth tests** with a fake token verifier and ensure no provider secret/token appears in responses or logs.

### Task 5: Port destination, attractions, weather, and routing services

**Files:**
- Create: `backend/internal/places/model.go`, `backend/internal/places/repository.go`, `backend/internal/places/service.go`, `backend/internal/places/handler.go`, corresponding `_test.go` files
- Create: `backend/internal/providers/geocoding.go`, `backend/internal/providers/weather.go`, `backend/internal/providers/routes.go`, corresponding `_test.go` files
- Modify: `backend/internal/httpapi/router.go`, `backend/internal/config/config.go`

**Interfaces:**
- Produces: `GET /api/v1/attractions?destination=&days=&date=&category=` returns stable day-grouped JSON used by the current selection flow.
- Produces: `GET /api/v1/attractions/{id}` returns one attraction; `POST /api/v1/routes` accepts ordered `{latitude,longitude}` waypoints and returns distance, duration, and GeoJSON route geometry from OSRM.

- [ ] **Step 1: Capture existing frontend response contract** in fixtures and write handler/service tests for invalid inputs, unknown destination, category filtering, no forecast, and weather-aware day grouping.
- [ ] **Step 2: Run targeted places/provider tests** and verify contract tests fail before implementation.
- [ ] **Step 3: Implement parameterized PostGIS queries** using actual legacy schema, with safe category filtering and stable attraction fields.
- [ ] **Step 4: Implement server-side OpenCage/OpenWeather clients** with request timeouts, validation, graceful no-key behavior, and no secret logging.
- [ ] **Step 5: Implement OSRM routing client** with input bounds, configurable endpoint, and actionable provider errors; use a fake HTTP server for provider tests.
- [ ] **Step 6: Run the places/provider test suite** and compare returned attraction shape to captured legacy fixtures.

### Task 6: Implement trip CRUD API and ownership protections

**Files:**
- Create: `backend/internal/trips/handler.go`, `backend/internal/trips/handler_test.go`
- Modify: `backend/internal/httpapi/router.go`, `backend/internal/trips/repository.go`

**Interfaces:**
- Produces: authenticated `POST /api/v1/trips`, `GET /api/v1/trips`, `GET /api/v1/trips/{id}`, `PUT /api/v1/trips/{id}`, `DELETE /api/v1/trips/{id}`.
- Request validation ensures valid dates/day count, ordered day assignments, known attractions, and bounded stop counts.

- [ ] **Step 1: Write handler tests** for CRUD, invalid dates/stop IDs, unauthenticated access, and cross-user read/update/delete attempts.
- [ ] **Step 2: Run targeted trip tests** and confirm failure before implementation.
- [ ] **Step 3: Implement trip handlers** with request size limits, validation, authenticated user context, and consistent JSON errors.
- [ ] **Step 4: Run trip handler integration tests** against PostgreSQL and verify no operation can cross user ownership.

### Task 7: Build Next.js discovery and authentication flows

**Files:**
- Create: `frontend/public/assets/` by carrying forward useful image and SVG assets from `Client/src/assets/` before deleting `Client/`
- Create: `frontend/src/lib/api.ts`, `frontend/src/lib/types.ts`, `frontend/src/lib/auth.ts`, `frontend/src/components/site-header.tsx`, `frontend/src/components/trip-search-form.tsx`, `frontend/src/components/auth-form.tsx`, `frontend/src/app/login/page.tsx`, `frontend/src/app/signup/page.tsx`, `frontend/src/app/auth/callback/page.tsx`
- Modify: `frontend/src/app/page.tsx`, `frontend/src/app/layout.tsx`, `frontend/src/app/globals.css`, `frontend/package.json`
- Delete after replacement: `Client/`, root legacy `package.json`/`package-lock.json`, `Server/`

**Interfaces:**
- Produces: typed API client configured via `NEXT_PUBLIC_API_BASE_URL`; accessible `/`, `/login`, `/signup`, and OAuth return handling.
- Form submission navigates to `/plan` with URL-encoded search inputs rather than relying on transient router state.

- [ ] **Step 1: Add frontend tests** for form validation, accessible labels/errors, API failure states, and OAuth return state.
- [ ] **Step 2: Run targeted frontend tests** and verify expected failures.
- [ ] **Step 3: Implement shared API types/client** and discovery/search/auth screens using server/client component boundaries intentionally.
- [ ] **Step 4: Match the approved visual direction**: editorial travel workspace, evergreen/mineral blue palette, humanist sans and serif display type, visible focus, reduced motion, mobile layout. Confirm the homepage and auth screens use the carried-forward travel imagery where it supports the brief.
- [ ] **Step 5: Run targeted frontend tests and `npm run build`** before removing the old client.

### Task 8: Build attraction selection, itinerary editing, and MapLibre views

**Files:**
- Create: `frontend/src/app/plan/page.tsx`, `frontend/src/app/trips/page.tsx`, `frontend/src/app/trips/[id]/page.tsx`, `frontend/src/components/planner-client.tsx`, `frontend/src/components/attraction-list.tsx`, `frontend/src/components/itinerary-editor.tsx`, `frontend/src/components/maplibre-map.tsx`, `frontend/src/components/route-summary.tsx`
- Modify: `frontend/src/lib/api.ts`, `frontend/src/lib/types.ts`, `frontend/src/app/globals.css`
- Remove Leaflet packages and styles from frontend dependencies.

**Interfaces:**
- Produces: destination planning, day assignment, drag/button-based stop reordering, route visualization, authenticated trip persistence, trip list/edit/delete, and sign-in prompt for anonymous save.
- MapLibre component is dynamically loaded only on map pages and receives configurable OpenFreeMap style and OSRM service URLs.

- [ ] **Step 1: Write UI tests** for loading/empty/error/anonymous/signed-in flows, reordering, day reassignment, save/reopen/delete, and route API failure.
- [ ] **Step 2: Run targeted UI tests** and confirm failure before implementation.
- [ ] **Step 3: Implement the accessible day/stop editor** with keyboard-operable move controls and persistent API mutations.
- [ ] **Step 4: Implement MapLibre markers, OSRM route geometry, fit bounds, and clear provider error states**; remove all Leaflet references/dependencies.
- [ ] **Step 5: Run UI tests and production build**, including manual narrow viewport review if available.

### Task 9: Bundle local development, migrations, and deployment configuration

**Files:**
- Create: `docker-compose.yml`, `Makefile`, root `.env.example`, `backend/Dockerfile`, `frontend/Dockerfile`
- Modify: `README.md`, `.gitignore`, root package scripts
- Delete: legacy `Client/` and `Server/` only after their required behavior has parity and no frontend references remain.

**Interfaces:**
- Produces: documented `make dev`, `make migrate`, `make test`, and production build commands; local PostgreSQL/PostGIS configuration and separate API/frontend environment files.

- [ ] **Step 1: Write a clean-checkout smoke checklist** covering migration, API readiness, frontend startup, password auth, Google OAuth setup, attraction lookup, maps, and trip CRUD.
- [ ] **Step 2: Implement Docker Compose and root commands** with secrets supplied through ignored env files and safe defaults.
- [ ] **Step 3: Rewrite README** with exact prerequisites, setup, migration, Google console callback/API setup, provider keys, deployment-origin/cookie guidance, and troubleshooting.
- [ ] **Step 4: Run the smoke checklist** with credentials omitted and with local PostgreSQL, then verify optional-provider screens remain usable.
- [ ] **Step 5: Search for stale Vite/Express/Leaflet/Mongo/TomTom credentials and paths**, remove obsolete dependencies/files, and run final backend/frontend builds and targeted tests.
