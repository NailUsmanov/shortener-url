package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DataBaseStorage struct {
	memory *MemoryStorage
	db     *sql.DB
}

func NewDataBaseStorage(dsn string) (*DataBaseStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &DataBaseStorage{db: db}, nil
}

func (d *DataBaseStorage) Save(url string) (string, error) {
	key, err := d.memory.Save(url)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (d *DataBaseStorage) Get(key string) (string, error) {
	url, exists := d.memory.data[key]
	if !exists {
		return "", fmt.Errorf("URL not found")
	}

	return url, nil
}

func (d *DataBaseStorage) Close() {
	d.db.Close()
}

func (d *DataBaseStorage) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
