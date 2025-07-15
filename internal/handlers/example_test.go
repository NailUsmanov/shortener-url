package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"go.uber.org/zap"
)

func ExampleNewCreateShortURL_correct() {
	// подготовка хранилища и логгера
	memStorage := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Создаем хендлер
	handler := handlers.NewCreateShortURL(memStorage, "http://localhost", logger.Sugar())

	// Создаем запрос
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("http://example.com"))
	req.Header.Set("Content-Type", "text/plain")

	// Подставляем userID в контекст
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)

	// Создаём ResponseRecorder
	rec := httptest.NewRecorder()

	// Вызываем хендлер
	handler.ServeHTTP(rec, req)

	// Печатаем результат
	fmt.Println("Status code:", rec.Code)

	// Output:
	// Status code: 201
}

func ExampleNewCreateShortURL_conflict() {
	// подготовка хранилища и логгера
	memStorage := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Создаем хендлер
	handler := handlers.NewCreateShortURL(memStorage, "http://localhost", logger.Sugar())

	url := "http://example.com"
	userID := "user123"

	// Создаем запрос (первый)
	req1 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
	req1.Header.Set("Content-Type", "text/plain")
	ctx1 := context.WithValue(context.Background(), middleware.UserIDKey, userID)
	req1 = req1.WithContext(ctx1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Второй запрос (конфликт)
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
	req2.Header.Set("Content-Type", "text/plain")
	ctx2 := context.WithValue(context.Background(), middleware.UserIDKey, userID)
	req2 = req2.WithContext(ctx2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	// Проверяем второй ответ
	fmt.Println("Status code:", rec2.Code)

	// Output:
	// Status code: 409
}
