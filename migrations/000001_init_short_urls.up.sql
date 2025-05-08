CREATE TABLE short_urls (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL UNIQUE
);

CREATE INDEX idx_original_url ON short_urls (original_url);