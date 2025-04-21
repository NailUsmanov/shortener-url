package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
)

type MemoryStorage struct {
	data     map[string]string
	mu       sync.RWMutex //Для потокобезопасности
	lastUUID int
	filePath string
}

func NewMemoryStorage(filePath string) *MemoryStorage {
	s := &MemoryStorage{
		data:     make(map[string]string),
		lastUUID: 0,
		filePath: filePath,
	}
	if filePath != "" {
		s.loadLastUUID()
		s.loadFromFile()
	}
	return s
}

func (s *MemoryStorage) loadFromFile() {
	file, err := os.Open(s.filePath)
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
		s.data[record.ShortURL] = record.OriginalURL
		if record.UUID > s.lastUUID {
			s.lastUUID = record.UUID
		}
	}
}

type ShortURLJSON struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (s *MemoryStorage) loadLastUUID() {
	if s.filePath == "" {
		s.lastUUID = 0
		return
	}
	file, err := os.Open("ListShortURL")
	if err != nil {
		if os.IsNotExist(err) {
			return // Файла нет - начинаем с 1
		}
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastRecord ShortURLJSON

	for scanner.Scan() {
		line := scanner.Bytes()
		var record ShortURLJSON
		if err := json.Unmarshal(line, &record); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			continue
		}
		if record.UUID > lastRecord.UUID {
			lastRecord = record
		}
	}
	s.lastUUID = lastRecord.UUID

}

func (s *MemoryStorage) Save(url string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := generateShortCode()
	s.data[key] = url

	if s.filePath != "" {
		file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return "", err
		}
		defer file.Close()

		s.lastUUID++
		ShortStr := ShortURLJSON{
			UUID:        s.lastUUID,
			ShortURL:    key,
			OriginalURL: url,
		}
		err = json.NewEncoder(file).Encode(ShortStr)
		if err != nil {
			s.lastUUID--
			return "", err
		}
	}
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

func generateShortCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)

	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}
