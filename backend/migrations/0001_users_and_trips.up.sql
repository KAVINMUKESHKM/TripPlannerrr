ALTER TABLE users ADD COLUMN IF NOT EXISTS google_subject TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS users_google_subject_unique
    ON users (google_subject)
    WHERE google_subject IS NOT NULL;

CREATE TABLE IF NOT EXISTS sessions (
    token_hash BYTEA PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS trips (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    destination_id INTEGER REFERENCES destinations(id) ON DELETE SET NULL,
    destination_name TEXT NOT NULL,
    start_date DATE NOT NULL,
    day_count SMALLINT NOT NULL CHECK (day_count BETWEEN 1 AND 30),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS trips_user_created_idx ON trips (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS trip_days (
    trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    day_number SMALLINT NOT NULL CHECK (day_number BETWEEN 1 AND 30),
    PRIMARY KEY (trip_id, day_number)
);

CREATE TABLE IF NOT EXISTS trip_stops (
    trip_id BIGINT NOT NULL,
    day_number SMALLINT NOT NULL,
    stop_position SMALLINT NOT NULL CHECK (stop_position >= 0),
    attraction_id INTEGER NOT NULL REFERENCES attractions(id) ON DELETE RESTRICT,
    PRIMARY KEY (trip_id, day_number, stop_position),
    UNIQUE (trip_id, day_number, attraction_id),
    FOREIGN KEY (trip_id, day_number)
        REFERENCES trip_days (trip_id, day_number) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS trip_stops_attraction_idx ON trip_stops (attraction_id);
