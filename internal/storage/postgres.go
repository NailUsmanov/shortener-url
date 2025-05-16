package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
)

type DataBaseStorage struct {
	db *sql.DB
}

var SelectShortURL string = "SELECT short_url FROM short_urls WHERE original_url = $1 AND user_id = $2"
var InsertOriginalAndShortURL string = "INSERT INTO short_urls (original_url, short_url, user_id) VALUES ($1, $2, $3)"
var PrepareSQL string = `INSERT INTO short_urls (original_url, short_url, user_id)
    VALUES ($1, $2, $3)
    ON CONFLICT (original_url) DO NOTHING
    RETURNING short_url`
var SelectOriginalURL string = `SELECT original_url FROM short_urls WHERE short_url = $1`
var SelectAllOriginalURL string = "SELECT short_url, original_url FROM short_urls WHERE user_id = $1"
var IS_DELETED_SQL string = "UPDATE short_urls SET is_deleted = true WHERE short_url = ANY($1) AND user_id = $2;"
var SelectOriginalURLWithFlag string = "SELECT original_url, is_deleted FROM shorts_urls WHERE short_url = $1"

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

	// Настройка миграций
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate driver: %w", err)
	}

	// Инициализация мигратора
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise migrate driver: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return &DataBaseStorage{db: db}, nil
}

func (d *DataBaseStorage) Save(ctx context.Context, url string, userID string) (string, error) {
	// Проверяем есть ли такая ссылка уже в базе данных и выдаем имеющийся ключ

	row := d.db.QueryRowContext(ctx, SelectShortURL, url, userID)
	var key string
	err := row.Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			// Генерация нового ключа
			key = generateShortCode()
			_, err = d.db.ExecContext(ctx, InsertOriginalAndShortURL, url, key, userID)
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
	var originalURL string
	var isDeleted bool

	row := d.db.QueryRowContext(ctx, SelectOriginalURLWithFlag, key)
	err := row.Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get URL: %v", err)
	}
	if isDeleted {
		return "", ErrDeleted
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

func (d *DataBaseStorage) SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error) {

	// Подготовка транзакции
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Подготовка SQL запроса
	stmt, err := tx.PrepareContext(ctx, PrepareSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %v", err)
	}

	defer stmt.Close()

	// Обработка SQL запроса
	var keys []string
	var conflictErr error
	for _, u := range urls {
		var key string
		err := stmt.QueryRowContext(ctx, u, generateShortCode(), userID).Scan(&key)
		if err == sql.ErrNoRows {

			// URL уже существует, получаем его ключ
			err = tx.QueryRowContext(ctx, SelectShortURL, u).Scan(&key)
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

func (d *DataBaseStorage) GetByURL(ctx context.Context, originalURL string, userID string) (string, error) {
	var shortURL string
	err := d.db.QueryRowContext(ctx,
		SelectShortURL, originalURL, userID).Scan(&shortURL)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	return shortURL, nil
}

func (d *DataBaseStorage) GetUserURLS(ctx context.Context, userID string) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	rows, err := d.db.QueryContext(ctx, SelectAllOriginalURL, userID)
	if err != nil {
		return nil, fmt.Errorf("db query: %v", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)

	for rows.Next() {
		var short, original string
		if err := rows.Scan(&short, &original); err != nil {
			return nil, fmt.Errorf("scan row: %v", err)
		}
		result[short] = original
	}
	return result, nil
}

func (d *DataBaseStorage) MarkAsDeleted(ctx context.Context, urls []string, userID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, err := d.db.ExecContext(ctx, IS_DELETED_SQL, pq.Array(urls), userID)
	if err != nil {
		log.Println("err with SQL request")
		return err
	}
	return nil
}
