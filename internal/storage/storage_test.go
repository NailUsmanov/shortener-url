package storage

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-storage-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	s := NewMemoryStorage(tmpFile.Name())

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
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		lines := strings.Split(string(data), "\n")
		var lastUUID int

		for _, line := range lines {
			if line == "" {
				continue
			}
			var record ShortURLJSON
			err := json.Unmarshal([]byte(line), &record)
			assert.NoError(t, err)
			assert.Equal(t, lastUUID+1, record.UUID)
			lastUUID = record.UUID
		}
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
}
