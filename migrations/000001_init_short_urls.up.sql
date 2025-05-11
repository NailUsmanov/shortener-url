CREATE TABLE short_urls (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL
);

CREATE INDEX idx_original_url ON short_urls (original_url);