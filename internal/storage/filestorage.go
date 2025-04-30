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
}

func (f *FileStorage) Save(url string) (string, error) {
	key, err := f.memory.Save(url)
	if err != nil {
		return "", err
	}

	if f.filePath != "" {
		f.saveToFile(key, url)

	}
	return key, nil
}

func (f *FileStorage) Get(key string) (string, error) {
	return f.memory.Get(key)
}

// Доп метод для сохранения в файл
func (f *FileStorage) saveToFile(key, url string) error {
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
		f.memory.data[record.ShortURL] = record.OriginalURL
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
