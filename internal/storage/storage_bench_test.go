package storage

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkSaveMemory(b *testing.B) {
	arrURL := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		arrURL[i] = "http://example.com/" + strconv.Itoa(i)
	}

	b.Run("save", func(b *testing.B) {
		mem := NewMemoryStorage()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			url := arrURL[i%1000]
			mem.Save(context.Background(), url+strconv.Itoa(i), strconv.Itoa(i))
		}
	})
}

func BenchmarkGetByURL(b *testing.B) {
	mem := NewMemoryStorage()

	// Подготовка данных: 1000 уникальных URL и userID
	urls := make([]string, 1000)
	userID := "42"

	for i := 0; i < len(urls); i++ {
		url := "http://example.com/" + strconv.Itoa(i)
		urls[i] = url
		_, err := mem.Save(context.Background(), url, userID)
		if err != nil {
			b.Fatalf("setup Save failed: %v", err)
		}
	}

	b.ResetTimer()

	// Бенчмаркинг
	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		_, err := mem.GetByURL(context.Background(), url, userID)
		if err != nil {
			b.Fatalf("GetByURL error: %v", err)
		}
	}
}

func BenchmarkGetUsersURLS(b *testing.B) {
	mem := NewMemoryStorage()
	urls := make([]string, 10000)
	userID := "11"

	for i := 0; i < len(urls); i++ {
		url := "http://example.com/" + strconv.Itoa(i)
		_, err := mem.Save(context.Background(), url, userID)
		if err != nil {
			b.Fatalf("setup Save failed: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := mem.GetUserURLS(context.Background(), userID)
		if err != nil {
			b.Fatalf("GetUsersURLs error: %v", err)
		}
	}

}
