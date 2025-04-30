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

func (s *MemoryStorage) Save(url string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := generateShortCode()
	s.data[key] = url

	return key, nil

}

func (s *MemoryStorage) Get(key string) (string, error) {
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
