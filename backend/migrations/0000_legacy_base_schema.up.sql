CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    full_name TEXT,
    bio TEXT
);

CREATE TABLE IF NOT EXISTS destinations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    best_climate VARCHAR(255),
    ideal_temp_min DECIMAL(5,2),
    ideal_temp_max DECIMAL(5,2),
    ideal_weather TEXT
);

CREATE TABLE IF NOT EXISTS attractions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    category VARCHAR(100),
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    destination_id INTEGER REFERENCES destinations(id),
    best_climate VARCHAR(255),
    ideal_temp_min DECIMAL(5,2),
    ideal_temp_max DECIMAL(5,2),
    ideal_weather TEXT
);
