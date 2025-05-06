package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DataBaseStorage struct {
	db *sql.DB
}

var ErrAlreadyHasKey = errors.New("key is exists")

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
		original_url TEXT NOT NULL UNIQUE,
		short_url TEXT NOT NULL UNIQUE
	);
`)

	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, "CREATE INDEX short_url_id ON shorts_urls (short_url_id)")
	if err != nil {
		return nil, err
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

	return key, ErrAlreadyHasKey // URL уже существует у нас в баще, возвращаем его short_url
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

func (d *DataBaseStorage) SaveInBatch(ctx context.Context, urls []string) ([]string, error) {

	// Подготовка транзакции
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Подготовка SQL запроса
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO short_urls (original_url, short_url)
    VALUES ($1, $2)
    ON CONFLICT (original_url) DO NOTHING
    RETURNING short_url`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %v", err)
	}

	defer stmt.Close()

	// Обработка SQL запроса
	var keys []string
	var conflictErr error
	for _, u := range urls {
		var key string
		err := stmt.QueryRowContext(ctx, u, generateShortCode()).Scan(&key)
		if err == sql.ErrNoRows {

			// URL уже существует, получаем его ключ
			err = tx.QueryRowContext(ctx, "SELECT short_url FROM short_urls WHERE original_url = $1", u).Scan(&key)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing URL: %v", err)
			}
			if conflictErr == nil {
				conflictErr = ErrAlreadyHasKey
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to save URL: %v", err)
		}

		keys = append(keys, key)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}
	return keys, nil

}

func (d *DataBaseStorage) GetByURL(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	err := d.db.QueryRowContext(ctx,
		"SELECT short_url FROM short_urls WHERE original_url = $1", originalURL).Scan(&shortURL)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	return shortURL, nil
}
