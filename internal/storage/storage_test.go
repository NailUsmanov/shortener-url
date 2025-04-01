package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
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
}
