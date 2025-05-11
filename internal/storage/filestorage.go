package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type FileStorage struct {
	memory    *MemoryStorage
	filePath  string
	lastUUID  int
	saveMutex sync.Mutex
}

func NewFileStorage(filePath string) *FileStorage {
	s := &FileStorage{
		memory:   NewMemoryStorage(),
		filePath: filePath,
	}
	s.loadFromFile()
	return s
}

type ShortURLJSON struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

func (f *FileStorage) Save(ctx context.Context, url string, userID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if key, err := f.GetByURL(ctx, url, userID); err == nil && key != "" {
		return key, ErrAlreadyHasKey
	}

	key, err := f.memory.Save(ctx, url, userID)
	if err != nil {
		return "", err
	}

	if f.filePath != "" {
		f.saveToFile(key, url, userID)

	}
	return key, nil
}

func (f *FileStorage) Get(ctx context.Context, key string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	return f.memory.Get(ctx, key)
}

// Доп метод для сохранения в файл
func (f *FileStorage) saveToFile(key, url string, userID string) error {
	f.saveMutex.Lock()
	defer f.saveMutex.Unlock()

	file, err := os.OpenFile(f.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	f.lastUUID++

	record := ShortURLJSON{
		UUID:        f.lastUUID,
		ShortURL:    key,
		OriginalURL: url,
		UserID:      userID,
	}
	return json.NewEncoder(file).Encode(record)
}

// Доп. метод для того, чтобы вытащить последнюю запись из файла и найти последний UUID
func (f *FileStorage) loadFromFile() {
	file, err := os.Open(f.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		var record ShortURLJSON
		if err := json.Unmarshal(line, &record); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			continue
		}
		f.memory.data[record.ShortURL] = URLData{
			OriginalURL: record.OriginalURL,
			UserID:      record.UserID,
		}
		if record.UUID > f.lastUUID {
			f.lastUUID = record.UUID
		}
	}
}

func (f *FileStorage) Ping(ctx context.Context) error {
	// Проверяем отмену контекста
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (f *FileStorage) SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error) {
	// Проверяем, не отменен ли контекст
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Заглушка: просто возвращаем фейковые ключи
	keys := make([]string, len(urls))
	for i := range keys {
		keys[i] = fmt.Sprintf("fake_key_%d", i) // Генерируем фейковый ключ
	}

	return keys, nil
}

func (f *FileStorage) GetByURL(ctx context.Context, originalURL string, userID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	f.memory.mu.RLock()
	defer f.memory.mu.RUnlock()

	for short, url := range f.memory.data {
		if url.OriginalURL == originalURL {
			if url.UserID != userID {
				return "", errors.New("User isnt find")
			}
			return short, nil
		}
	}
	return "", errors.New("URL isn't find")
}

func (f *FileStorage) GetUserURLS(ctx context.Context, userID string) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result := make(map[string]string)
	f.memory.mu.RLock()
	defer f.memory.mu.RUnlock()

	for short, data := range f.memory.data {
		if data.UserID == userID {
			result[short] = data.OriginalURL
		}
	}

	return result, nil
}
