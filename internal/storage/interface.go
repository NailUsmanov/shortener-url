package storage

import "context"

type Storage interface {
	Save(ctx context.Context, url string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
}
