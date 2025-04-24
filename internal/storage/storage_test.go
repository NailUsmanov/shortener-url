package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Для хранения в мап
func TestInMemoryStorage(t *testing.T) {

	s := NewMemoryStorage()
	t.Run("Save and Get", func(t *testing.T) {
		url := "http://example.com"
		key, err := s.Save(url)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)

		val, err := s.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, url, val)

	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := s.Get("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Generate short code uniqueness", func(t *testing.T) {
		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code := generateShortCode()
			assert.False(t, codes[code], "Duplicate code generated")
			codes[code] = true
		}
	})
}

// Для хранения в файле

func TestFileStorage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-storage-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Подготовка тестовых данных
	testRecords := []ShortURLJSON{
		{UUID: 1, ShortURL: "test123", OriginalURL: "http://test.com"},
		{UUID: 2, ShortURL: "test456", OriginalURL: "http://another.com"},
	}

	for _, r := range testRecords {
		data, err := json.Marshal(r)
		require.NoError(t, err)
		_, err = tmpFile.WriteString(string(data) + "\n")
		require.NoError(t, err)

	}
	tmpFile.Close()

	t.Run("Initialization and load from file", func(t *testing.T) {
		s := NewFileStorage(tmpFile.Name())

		val, err := s.Get("test123")
		assert.NoError(t, err)
		assert.Equal(t, "http://test.com", val)

		val, err = s.Get("test456")
		assert.NoError(t, err)
		assert.Equal(t, "http://another.com", val)

		assert.Equal(t, 2, s.lastUUID)

	})

	t.Run("Save New URL", func(t *testing.T) {
		s := NewFileStorage(tmpFile.Name())
		url := "http://new-example.com"
		key, err := s.Save(url)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)

		//Проверка сохранения в памяти
		val, err := s.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, url, val)

		//Проверка записи в файл
		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		var found bool
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var record ShortURLJSON
			err := json.Unmarshal(scanner.Bytes(), &record)
			require.NoError(t, err)
			if record.OriginalURL == url {
				found = true
				break
			}
		}
		assert.True(t, found, "New URL not found in file")
	})

	t.Run("Load from non-existent file", func(t *testing.T) {
		nonExistentFile := "non-existent-file.json"
		defer os.Remove(nonExistentFile)

		s := NewFileStorage(nonExistentFile)
		assert.NotNil(t, s)

		// Проверяем что можем сохранять/получать несмотря на отсутствие файла
		key, err := s.Save("http://new-url.com")
		assert.NoError(t, err)

		val, err := s.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, "http://new-url.com", val)
	})
}
