package storage

import (
	"context"
	"database/sql"
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

	return &DataBaseStorage{db: db,
		memory: NewMemoryStorage()}, nil
}

func (d *DataBaseStorage) Save(url string) (string, error) {
	return d.memory.Save(url)
}

func (d *DataBaseStorage) Get(key string) (string, error) {
	return d.memory.Get(key)
}

func (d *DataBaseStorage) Close() {
	d.db.Close()
}

func (d *DataBaseStorage) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
