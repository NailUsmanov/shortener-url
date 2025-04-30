package storage

import "context"

type Storage interface {
	Save(url string) (string, error)
	Get(key string) (string, error)
	Ping(ctx context.Context) error
}
