package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyHasKey = errors.New("url already exists")

type BasicStorage interface {
	Save(ctx context.Context, url string, userID string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
}

type BatchStorage interface {
	SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error)
}

type URLFinder interface {
	GetByURL(ctx context.Context, url string, userID string) (string, error)
	GetUserURLS(ctx context.Context, userID string) (map[string]string, error)
}

type Storage interface {
	BasicStorage
	BatchStorage
	URLFinder
}
