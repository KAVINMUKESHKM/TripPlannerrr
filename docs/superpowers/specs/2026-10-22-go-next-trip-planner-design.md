# Trip Planner Go and Next.js Migration Design

## Intent

Replace the existing Express/Vite implementation with a maintainable Go API and a Next.js web app while preserving the trip planning journey. The app helps travelers discover weather-aware attractions, choose stops, and understand how to get between them. Add Google OAuth and saved, editable itineraries so plans persist across visits.

## Goals

- Replace the JavaScript backend with Go 1.27.3 and expose documented, versioned JSON APIs.
- Replace the Vite client with the latest stable Next.js release at implementation time (currently 16.4.2), keeping the existing Trip Planner brand and useful travel content while improving the interface.
- Keep PostgreSQL/PostGIS as the source for users, destinations, and attractions; remove the unused MongoDB integration.
- Retain destination search, travel dates and duration, weather-aware recommendations, attraction selection, and route display.
- Replace Leaflet with MapLibre GL JS using OpenFreeMap styles/tiles, and use OSRM for route geometry/distance/duration.
- Support password registration/login and Google OAuth sign-in, with secure server-managed sessions.
- Let authenticated users save, edit, reopen, and delete their itineraries, including destination, date range, selected attractions, and ordering by day.
- Provide responsive, accessible loading, empty, error, and unauthenticated states.
- Bundle the API and frontend source, configuration examples, database migrations, and local-development instructions in this repository.

## Non-goals

- Booking, payments, social feeds, user reviews, or real-time collaboration.
- Migrating to a different database or supporting MongoDB.
- Guaranteeing third-party weather/places data availability when API keys or provider quotas are absent.
- Reproducing the current UI pixel-for-pixel; preserve product identity and task flow while improving usability.

## Constraints and assumptions

- Go version is exactly `1.27.3`; use the latest stable Next.js version available when dependencies are installed.
- PostgreSQL is already in use and attraction rows include coordinates and weather suitability fields. Use SQL migrations rather than creating schema as a side effect of server startup.
- Use Google as the OAuth provider. OAuth credentials are supplied through environment variables; no secret is committed.
- Keep existing useful attraction data and assets. Inspect and preserve the real schema/data contract before designing migrations.
- Itinerary ownership is per signed-in user. Password-authenticated accounts remain supported alongside OAuth accounts.
- MapLibre renders maps from OpenFreeMap styles/tiles and OSRM supplies routes. Keep the providers configurable so deployment can select its own compatible endpoints; show a clear map/routing error if a service is unavailable.
- Do not make users configure paid APIs just to load non-map pages; document which keys unlock which functions.

## Product flow

1. A traveler enters destination, start date, trip length, and optionally attraction category.
2. The API geocodes the destination, reads matching PostGIS attractions, and combines them with available forecast data to return day-grouped suggestions.
3. The traveler selects/reorders attractions across days and views the itinerary on an interactive Google Map with route distance and duration.
4. A signed-in traveler saves the plan; they can reopen, edit, or delete it from a trips view. An anonymous traveler can explore and is prompted to sign in when saving.
5. Sign-in is available through existing email/password registration and login or Google OAuth.

## Architecture

- `backend/`: Go HTTP API, configuration, PostgreSQL repository, domain services, Google OAuth callback, cookie session middleware, provider integrations, and SQL migrations. Keep handlers thin and repositories explicit. Health/readiness endpoints expose no secrets.
- `frontend/`: Next.js App Router application, server/client components split by interaction needs, typed API client, auth-aware navigation, itinerary editor, and dynamically loaded MapLibre components.
- PostgreSQL: users with optional Google subject, OAuth state/session storage as appropriate, and normalized user-owned trips/days/stops. Existing destinations and attractions remain compatible.
- External providers: Google OAuth, OpenFreeMap, MapLibre, OSRM, OpenCage geocoding, and OpenWeather forecasts. Routing and geocoding calls remain server-side; the map uses public style/tile endpoints that can be configured.
- Bundle: one repository with independently deployable Go and Next.js applications, root development/build commands, `.env.example` files, migration command, and Docker Compose for local PostgreSQL where practical.

## Security and reliability

- OAuth uses authorization code flow, state and nonce validation, exact redirect URI configuration, and secure, HttpOnly, SameSite cookies.
- Use parameterized SQL, password hashing, request validation, bounded request bodies, timeouts, and provider error handling.
- Production cookies are Secure; local development behavior is explicitly configured.
- Never return password hashes or provider tokens to the browser. Protect itinerary endpoints by authenticated user ID.
- Keep provider credentials out of source control; no browser map API key is required for the configured public map style.

## Visual direction

Keep the existing travel-led identity and destination imagery, replacing the current dense forms and oversized attraction lists with a calm, editorial trip workspace. Use deep evergreen and mineral blue as restrained functional colors, warm paper surfaces, and a citrus accent for selected stops. Use a distinctive humanist sans for interface text paired with a sturdy serif for destination headlines. The itinerary page should place a readable day-by-day stop list beside a persistent map on wide screens and stack them on mobile. Keep copy short, sentence case, and action-specific; prioritize visible keyboard focus and reduced-motion support.

## Open implementation choices for the builder

- Choose Go HTTP/router and PostgreSQL libraries that fit the repository and have active support; keep dependencies modest.
- Choose session storage and OAuth account-linking behavior consistent with the existing users table and deployment constraints.
- Use existing provider integrations where sound, repairing or replacing those that expose secrets, rely on brittle behavior, or have no useful tests.
- Add the smallest useful planning enhancement beyond persistence: reorder stops within a day and assign saved stops to a day.

## Acceptance criteria

- A clean checkout can run the Go API, Next.js app, PostgreSQL, and migrations from documented instructions.
- Existing destination-to-attractions flow remains available through the new app and API.
- Maps render with MapLibre using OpenFreeMap styles/tiles and route details come from OSRM; no Leaflet dependency remains.
- Email/password and Google OAuth sign-in establish secure sessions; logout clears them.
- Signed-in users can create, list, update, and delete only their own itineraries, and saved stops/days persist after reload.
- Missing provider credentials produce actionable setup errors while health and non-dependent screens remain usable.
- API, database, auth, and core frontend flows have targeted automated coverage and production builds succeed.
