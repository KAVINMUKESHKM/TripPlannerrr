# Trip Planner

Trip Planner helps people find attractions, arrange day-by-day itineraries, see their route on a map, and save trips to an account. The current application uses a Go API and a Next.js frontend.

## Current applications

- [Backend guide](backend/README.md) — Go API, database setup, migrations, imported place data, and API routes.
- [Frontend guide](frontend/README.md) — Next.js setup, pages, environment variables, and UI behavior.

The former `Client/` and `Server/` applications have been retired. Their active features are carried by the Go and Next.js applications:

| Previous app | Current app |
| --- | --- |
| React/Vite destination search and attraction list | Next.js home and trip planner |
| Attraction details and nearby restaurant suggestions | Attraction cards with expandable nearby restaurant results |
| Leaflet shortest-path map | MapLibre map with OSRM route line, distance, and duration |
| Register, login, logout | Go session auth, plus Google OAuth |
| Weather-matched attraction suggestions | Ideal-weather preferences filter suggestions when forecast data is available; forecasts remain visible by itinerary day |
| — | Saved trips with create, edit, and delete |

## Run locally

Use two terminals from the repository root.

### 1. Backend

If `backend/.env` does not exist, copy `backend/.env.example` to `backend/.env` and set `DATABASE_URL`. The existing local `.env` is left untouched.

```bash
cd backend
go run ./cmd/migrate up
go run ./cmd/importosm   # optional; loads the bundled Tamil Nadu attractions
go run ./cmd/api
```

### 2. Frontend

If `frontend/.env.local` does not exist, create it from `frontend/.env.example`. Then run:

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The frontend proxies `/api/*` to `http://localhost:8080` by default.

Root convenience commands are available as `npm run dev:backend`, `npm run dev:frontend`, `npm run build:frontend`, and `npm run test:backend`.

## Optional providers

The core app runs without weather, geocoding, Google OAuth, or restaurant credentials. OpenCage is needed for destinations that are not in the local database. OpenWeather adds daily forecasts. Google OAuth requires its client ID, secret, and registered callback. SerpAPI enables nearby restaurants. OpenFreeMap and OSRM provide map tiles and driving routes.

Provider variables and callback setup are documented in the app-specific READMEs. Keep real `.env` files private; use the `.env.example` files as templates.

## Place data

The bundled Tamil Nadu attraction records come from OpenStreetMap and are imported by `backend/cmd/importosm`. See [the data notes](backend/data/README.md) for source and attribution details.
