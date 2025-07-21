package storage

import (
	"context"
	"errors"
)

// Типизированные ошибки, используемые при работе с хранилищем URL.
var (
	// ErrNotFound возникает, если сокращённый URL не найден.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyHasKey возникает, если оригинальный URL уже существует в системе.
	ErrAlreadyHasKey = errors.New("url already exists")

	// ErrDeleted означает, что URL был помечен как удалённый.
	ErrDeleted = errors.New("url deleted")
)

// BasicStorage определяет базовые операции сохранения и получения URL.
type BasicStorage interface {
	Save(ctx context.Context, url string, userID string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
}

// BatchStorage описывает возможность пакетного сохранения нескольких URL.
type BatchStorage interface {
	SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error)
}

// URLFinder определяет методы поиска URL по оригинальному адресу или по ID пользователя.
type URLFinder interface {
	GetByURL(ctx context.Context, url string, userID string) (string, error)
	GetUserURLS(ctx context.Context, userID string) (map[string]string, error)
}

// URLDeleter описывает возможность для удаления URL из памяти.
type URLDeleter interface {
	MarkAsDeleted(ctx context.Context, urls []string, userID string) error
}

// Storage объединяет все интерфейсы для работы с сокращёнными URL.
type Storage interface {
	BasicStorage
	BatchStorage
	URLFinder
	URLDeleter
}
