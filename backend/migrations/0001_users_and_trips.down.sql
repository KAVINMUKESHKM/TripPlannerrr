DROP TABLE IF EXISTS trip_stops;
DROP TABLE IF EXISTS trip_days;
DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS sessions;
DROP INDEX IF EXISTS users_google_subject_unique;
ALTER TABLE users DROP COLUMN IF EXISTS google_subject;
