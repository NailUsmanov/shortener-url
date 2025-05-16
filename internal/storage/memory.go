package storage

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
)

type MemoryStorage struct {
	data map[string]URLData
	mu   sync.RWMutex //Для потокобезопасности
}

type URLData struct {
	OriginalURL string
	UserID      string
	Deleted     bool
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]URLData),
	}
}

func (s *MemoryStorage) Save(ctx context.Context, url string, userID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for short, original := range s.data {
		if original.OriginalURL == url {
			return short, ErrAlreadyHasKey // Возвращаем существующий ключ
		}
	}
	key := generateShortCode()
	s.data[key] = URLData{
		OriginalURL: url,
		UserID:      userID,
	}
	return key, nil

}

func (s *MemoryStorage) Get(ctx context.Context, key string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.data[key]
	if !exists {
		return "", fmt.Errorf("url not found")
	}

	if url.Deleted {
		return "", ErrDeleted
	}

	return url.OriginalURL, nil
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

func (s *MemoryStorage) SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	keys := make([]URLData, len(urls))
	result := make([]string, len(urls))
	for i := range keys {
		key := generateShortCode()
		s.data[key] = URLData{
			OriginalURL: urls[i], // Генерируем фейковый ключ
			UserID:      userID,
		}
		result[i] = key
	}

	return result, nil
}

func (s *MemoryStorage) GetByURL(ctx context.Context, OriginalURL string, userID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for shortURL, url := range s.data {
		if url.OriginalURL == OriginalURL && url.UserID == userID {
			if !url.Deleted {
				return shortURL, nil
			}
		}
	}
	return "", nil
}

func (s *MemoryStorage) GetUserURLS(ctx context.Context, userID string) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	AllURLS := map[string]string{}
	for short, url := range s.data {
		if url.UserID == userID && !url.Deleted {
			AllURLS[short] = url.OriginalURL
		}
	}
	return AllURLS, nil
}

func (s *MemoryStorage) MarkAsDeleted(ctx context.Context, urls []string, userID string) error {
	for _, shortURL := range urls {
		data, exists := s.data[shortURL]
		if exists && data.UserID == userID {
			data.Deleted = true
			s.data[shortURL] = data
		} else {
			return errors.New("err not found")
		}
	}
	return nil
}
