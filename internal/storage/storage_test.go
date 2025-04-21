package storage

import (
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
		newStorage := NewMemoryStorage(tmpFile.Name())

		url := "http://example.com"
		val, err := newStorage.Get("expected_key")
		assert.NoError(t, err)
		assert.Equal(t, url, val)
	})
}
