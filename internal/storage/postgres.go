package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DataBaseStorage struct {
	db *sql.DB
}

func NewDataBaseStorage(dsn string) (*DataBaseStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %v", err)
	}

	// Создаем таблицу, если ее нет
	_, err = db.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS short_urls (
		id SERIAL PRIMARY KEY,
		original_url TEXT NOT NULL,
		short_url TEXT NOT NULL UNIQUE
	);
	CREATE INDEX IF NOT EXISTS idx_original_url ON short_urls(original_url);
	CREATE INDEX IF NOT EXISTS idx_short_url ON short_urls(short_url);
`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	return &DataBaseStorage{db: db}, nil
}

func (d *DataBaseStorage) Save(ctx context.Context, url string) (string, error) {
	// Проверяем есть ли такая ссылка уже в базе данных и выдаем имеющийся ключ
	row := d.db.QueryRowContext(ctx, "SELECT short_url FROM short_urls WHERE original_url = $1", url)
	var key string
	err := row.Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			// Генерация нового ключа
			key = generateShortCode()
			_, err = d.db.ExecContext(ctx, `INSERT INTO short_urls (original_url, short_url) VALUES ($1, $2)`, url, key)
			if err != nil {
				return "", fmt.Errorf("failed to save URL: %v", err)
			}
			return key, nil
		}
		return "", fmt.Errorf("failed to check URL existence: %v", err) //  Возвращаю ошибку, если это не ErrNoRows
	}
	return key, nil // URL уже существует у нас в баще, возвращаем его short_url
}

func (d *DataBaseStorage) Get(ctx context.Context, key string) (string, error) {

	row := d.db.QueryRowContext(ctx, `SELECT original_url FROM short_urls WHERE short_url = $1`, key)

	var originalURL string
	err := row.Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("failed to get URL: %v", err)
	}
	return originalURL, nil
}

func (d *DataBaseStorage) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *DataBaseStorage) Ping(ctx context.Context) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return d.db.PingContext(ctx)
}
