package storage

import "context"

type Storage interface {
	SaveInBatch(ctx context.Context, urls []string) ([]string, error)
	Save(ctx context.Context, url string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
	GetByURL(ctx context.Context, url string) (string, error)
}
