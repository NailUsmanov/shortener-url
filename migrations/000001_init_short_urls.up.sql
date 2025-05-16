CREATE TABLE short_urls (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL
);

ALTER TABLE short_urls ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_original_url ON short_urls (original_url);