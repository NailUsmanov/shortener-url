package storage

import (
	"bufio"
	"context"
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
		userID := "user1"
		key, err := s.Save(context.Background(), url, userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)

		val, err := s.Get(context.Background(), key)
		assert.NoError(t, err)
		assert.Equal(t, url, val)

	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := s.Get(context.Background(), "nonexistent")
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

	t.Run("Get By URL", func(t *testing.T) {
		s := NewMemoryStorage()
		url := "http://example.com"
		userID := "user1"

		shortURL, err := s.Save(context.Background(), url, userID)
		require.NoError(t, err)

		_, err = s.Save(context.Background(), url, userID)
		assert.ErrorIs(t, err, ErrAlreadyHasKey)

		key, err := s.GetByURL(context.Background(), url, userID)
		assert.NoError(t, err)
		assert.Equal(t, shortURL, key)

	})

	t.Run("Non-existent URL", func(t *testing.T) {
		s := NewMemoryStorage()
		url := "http://example.com"
		userID := "user1"
		key, err := s.GetByURL(context.Background(), url, userID)
		assert.NoError(t, err)
		assert.Empty(t, key)
	})

	t.Run("Cancelled context", func(t *testing.T) {
		s := NewMemoryStorage()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		userID := "user1"
		_, err := s.GetByURL(ctx, "http://example.com", userID)
		assert.ErrorIs(t, err, context.Canceled)
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
		s, err := NewFileStorage(tmpFile.Name())
		assert.NoError(t, err)

		val, err := s.Get(context.Background(), "test123")
		assert.NoError(t, err)
		assert.Equal(t, "http://test.com", val)

		val, err = s.Get(context.Background(), "test456")
		assert.NoError(t, err)
		assert.Equal(t, "http://another.com", val)

		assert.Equal(t, 2, s.lastUUID)

	})

	t.Run("Save New URL", func(t *testing.T) {
		s, _ := NewFileStorage(tmpFile.Name())
		userID := "user1"
		url := "http://new-example.com"
		key, err := s.Save(context.Background(), url, userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)

		//Проверка сохранения в памяти
		val, err := s.Get(context.Background(), key)
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

		s, err := NewFileStorage(nonExistentFile)
		assert.NoError(t, err)
		assert.NotNil(t, s)

		// Проверяем что можем сохранять/получать несмотря на отсутствие файла
		userID := "user1"
		key, err := s.Save(context.Background(), "http://new-url.com", userID)
		assert.NoError(t, err)

		val, err := s.Get(context.Background(), key)
		assert.NoError(t, err)
		assert.Equal(t, "http://new-url.com", val)
	})
}

func TestPostgresStorage(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping PostgreSQL tests")
	}

	s, err := NewDataBaseStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	// Cleanup before tests
	ctx := context.Background()
	_, err = s.db.ExecContext(ctx, "DELETE FROM short_urls")
	require.NoError(t, err)

	t.Run("Save and Get", func(t *testing.T) {
		url := "http://example.com"
		userID := "user1"
		key, err := s.Save(ctx, url, userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)

		val, err := s.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, url, val)
	})

	t.Run("Save duplicate URL", func(t *testing.T) {
		url := "http://duplicate.com"
		userID := "user1"
		key1, err := s.Save(ctx, url, userID)
		assert.NoError(t, err)

		userID2 := "user2"
		key2, err := s.Save(ctx, url, userID2)
		assert.NoError(t, err)
		assert.Equal(t, key1, key2, "Should return same key for same URL")
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := s.Get(ctx, "nonexistent")
		assert.Error(t, err)
	})

	t.Run("Ping", func(t *testing.T) {
		err := s.Ping(ctx)
		assert.NoError(t, err)
	})
}

func TestPostgresStorage_SaveInBatch(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping PostgreSQL tests")
	}
	s, err := NewDataBaseStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	ctx := context.Background()
	_, err = s.db.ExecContext(ctx, "TRUNCATE TABLE short_urls")
	require.NoError(t, err)

	urls := []string{
		"http://example.com/batch1",
		"http://example.com/batch2",
		"http://example.com/batch1", // Дубликат
	}

	t.Run("Save batch with duplicates", func(t *testing.T) {
		userID := "user1"
		keys, err := s.SaveInBatch(ctx, urls, userID)
		assert.NoError(t, err)
		assert.Len(t, keys, len(urls))
		assert.Equal(t, keys[0], keys[2], "Duplicate URLs should return same keys")

		// Проверка что все URL доступны
		for i, key := range keys {
			val, err := s.Get(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, urls[i], val)
		}
	})
}
