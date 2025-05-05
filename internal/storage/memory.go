package storage

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
)

type MemoryStorage struct {
	data map[string]string
	mu   sync.RWMutex //Для потокобезопасности
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]string),
	}
}

func (s *MemoryStorage) Save(ctx context.Context, url string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := generateShortCode()
	s.data[key] = url

	return key, nil

}

func (s *MemoryStorage) Get(ctx context.Context, key string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	url, exists := s.data[key]
	if !exists {
		return "", fmt.Errorf("URL not found")
	}

	return url, nil
}

func (s *MemoryStorage) Ping(ctx context.Context) error {
	return ctx.Err()
}

func generateShortCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)

	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func (s *MemoryStorage) SaveInBatch(ctx context.Context, urls []string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	keys := make([]string, len(urls))
	for i := range keys {
		keys[i] = fmt.Sprintf("fake_key_%d", i) // Генерируем фейковый ключ
	}

	return keys, nil
}

func (s *MemoryStorage) GetByURL(ctx context.Context, OriginalURL string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortURL, url := range s.data {
		if url == OriginalURL {
			return shortURL, nil
		}
	}
	return "", nil
}
