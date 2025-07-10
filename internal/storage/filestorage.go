package storage

import (
	"bufio"
	"context"
	"encoding/json"
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

func NewFileStorage(filePath string) (*FileStorage, error) {
	s := &FileStorage{
		memory:   NewMemoryStorage(),
		filePath: filePath,
	}
	if filePath != "" {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
				return nil, fmt.Errorf("cannot create storage file: %w", err)
			}
		}
		s.loadFromFile()
	}
	return s, nil

}

type ShortURLJSON struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

// Используется для сохранения УРЛ в файл
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
		fmt.Printf("Memory save error: %v\n", err)
		return "", err
	}

	if f.filePath != "" {
		if err := f.saveToFile(key, url, userID); err != nil {
			fmt.Printf("File save error: %v\n", err) // Логируем ошибку записи
			return "", fmt.Errorf("failed to save to file: %w", err)
		}

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

	// 1. Открытие файла с правильными флагами
	file, err := os.OpenFile(f.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("file open error: %w", err)
	}
	defer file.Close()

	f.lastUUID++
	record := ShortURLJSON{
		UUID:        f.lastUUID,
		ShortURL:    key,
		OriginalURL: url,
		UserID:      userID,
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}
	// 4. Синхронизация записи
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync error: %w", err)
	}
	return nil
}

// Доп. метод для того, чтобы вытащить последнюю запись из файла и найти последний UUID
func (f *FileStorage) loadFromFile() {
	file, err := os.Open(f.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("error opening file: %v\n", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return
	}
	if fileInfo.Size() == 0 {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Пропускаем пустые строки
		}
		var record ShortURLJSON
		if err := json.Unmarshal(line, &record); err != nil {
			fmt.Printf("error parsing JSON: %v\n", err)
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
				return "", nil
			}
			return short, nil
		}
	}
	return "", ErrNotFound
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

func (f *FileStorage) MarkAsDeleted(ctx context.Context, urls []string, userID string) error {
	return nil
}
