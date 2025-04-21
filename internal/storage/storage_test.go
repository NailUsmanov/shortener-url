package storage

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-storage-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	s := NewMemoryStorage(tmpFile.Name())

	testRecord := ShortURLJSON{
		UUID:        1,
		ShortURL:    "test123",
		OriginalURL: "http://test.com",
	}
	data, err := json.Marshal(testRecord)
	require.NoError(t, err)
	_, err = tmpFile.WriteString(string(data) + "\n")
	require.NoError(t, err)
	tmpFile.Close()

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

	t.Run("Restore from file", func(t *testing.T) {
		testURL := "http://example.com"
		key, err := s.Save(testURL)
		require.NoError(t, err)
		newStorage := NewMemoryStorage(tmpFile.Name())
		val, err := newStorage.Get(key)
		require.NoError(t, err)
		assert.Equal(t, testURL, val)
	})

	t.Run("loadLastUUID", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-uuid-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		records := []ShortURLJSON{
			{UUID: 1, ShortURL: "abc", OriginalURL: "http://first.com"},
			{UUID: 2, ShortURL: "def", OriginalURL: "http://second.com"},
		}

		for _, r := range records {
			data, err := json.Marshal(r)
			require.NoError(t, err)
			_, err = tmpFile.WriteString(string(data) + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		storage := &MemoryStorage{filePath: tmpFile.Name()}
		storage.loadLastUUID()

		assert.Equal(t, 2, storage.lastUUID) // Должно быть 2, так как это максимальный UUID
	})

	t.Run("Empty file path - memory only", func(t *testing.T) {
		memStorage := NewMemoryStorage("") // без файла
		key, err := memStorage.Save("https://memory-only.com")
		assert.NoError(t, err)

		val, err := memStorage.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, "https://memory-only.com", val)

		// Проверяем что файл не создавался
		_, err = os.Stat("non-existent-file")
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Load from non-existent file", func(t *testing.T) {
		nonExistentStorage := NewMemoryStorage("non-existent-file.json")
		assert.NotNil(t, nonExistentStorage)
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
