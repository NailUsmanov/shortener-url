// Package storage описывает хранилище для сокращенных URL.
// В зависимости от выбора пользователя сохранение может быть в память, файл или postgre.
package storage

import (
	"context"
	"errors"
	"math/rand"
	"sync"
)

// MemoryStorage — in-memory хранилище сокращённых URL.
// Использует мапу и мьютекс для потокобезопасного доступа.
type MemoryStorage struct {
	data map[string]URLData
	mu   sync.RWMutex //Для потокобезопасности
}

// URLData содержит информацию об оригинальном URL, ID пользователя и флаг удаления.
type URLData struct {
	OriginalURL string
	UserID      string
	Deleted     bool
}

// NewMemoryStorage создает новое in-memory хранилище URL.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]URLData),
	}
}

// Save сохраняет оригинальный URL и его сокращение в память.
//
// Если URL уже существует — возвращает уже существующий короткий ключ.
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

// Get выдает полный URL по его сокращенному варианту.
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
		return "", ErrNotFound
	}

	if url.Deleted {
		return "", ErrDeleted
	}

	return url.OriginalURL, nil
}

// MemoryStorage.Ping используется для проверки соединения с БД.
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

// SaveInBatch позволяет сократить и сохранить в базу сразу несколько URL.
//
// Возвращает срез сокращенных URL
func (s *MemoryStorage) SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	keys := make([]URLData, len(urls))
	result := make([]string, len(urls))
	for i := range keys {
		key := generateShortCode()
		s.data[key] = URLData{
			OriginalURL: urls[i], // Генерируем уникальный ключ.
			UserID:      userID,
		}
		result[i] = key
	}

	return result, nil
}

// GetByURL позволяет получить сокращенный URL по его оригиналу.
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

// GetUserURLS выдает все пары (сокращенные URL и его оригинал), отправленные  когда-либо пользователем.
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

// MarkAsDeleted помечает URL для удаления в фоновом выполнении
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
