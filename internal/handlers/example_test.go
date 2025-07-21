package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
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

func ExampleNewRedirect_correct() {
	stor := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handlerRedirect := handlers.NewRedirect(stor, logger.Sugar())
	handlerCreate := handlers.NewCreateShortURL(stor, "http://localhost", logger.Sugar())
	// для записи в базу данных и дальнейшего поиска в базе для редиректа
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("http://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handlerCreate.ServeHTTP(rec, req)

	// получаем id из тела ответа
	shortURL := rec.Body.String()
	id := strings.TrimPrefix(shortURL, "http://localhost/")

	r := chi.NewRouter()
	r.Get("/{id}", handlerRedirect)

	reqRedir := httptest.NewRequest(http.MethodGet, "/"+id, bytes.NewBufferString("http://example.com"))
	ctxRedir := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	reqRedir = reqRedir.WithContext(ctxRedir)
	recRedir := httptest.NewRecorder()
	r.ServeHTTP(recRedir, reqRedir)

	fmt.Println("Status code:", recRedir.Code)
	fmt.Println("Location:", recRedir.Header().Get("Location"))
	// Output:
	// Status code: 307
	// Location: http://example.com

}

func ExampleNewRedirect_not_found() {
	stor := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handlerRedirect := handlers.NewRedirect(stor, logger.Sugar())
	// получаем id из тела ответа
	id := "test"

	r := chi.NewRouter()
	r.Get("/{id}", handlerRedirect)

	reqRedir := httptest.NewRequest(http.MethodGet, "/"+id, bytes.NewBufferString("http://example.com"))
	ctxRedir := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	reqRedir = reqRedir.WithContext(ctxRedir)
	recRedir := httptest.NewRecorder()
	r.ServeHTTP(recRedir, reqRedir)

	fmt.Println("Status code:", recRedir.Code)
	fmt.Println("Location:", recRedir.Header().Get("Location"))
	// Output:
	// Status code: 404
	// Location:

}

func ExampleNewRedirect_deleted() {
	stor := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handlerRedirect := handlers.NewRedirect(stor, logger.Sugar())
	handlerCreate := handlers.NewCreateShortURL(stor, "http://localhost", logger.Sugar())
	// для записи в базу данных и дальнейшего поиска в базе для редиректа
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("http://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handlerCreate.ServeHTTP(rec, req)

	// получаем id из тела ответа
	shortURL := rec.Body.String()
	id := strings.TrimPrefix(shortURL, "http://localhost/")

	// удаляем URL из базы
	err := stor.MarkAsDeleted(context.Background(), []string{id}, "user123")
	if err != nil {
		panic("mark as deleted failed: " + err.Error())
	}

	r := chi.NewRouter()
	r.Get("/{id}", handlerRedirect)

	reqRedir := httptest.NewRequest(http.MethodGet, "/"+id, bytes.NewBufferString("http://example.com"))
	ctxRedir := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	reqRedir = reqRedir.WithContext(ctxRedir)
	recRedir := httptest.NewRecorder()
	r.ServeHTTP(recRedir, reqRedir)

	fmt.Println("Status code:", recRedir.Code)
	fmt.Println("Location:", recRedir.Header().Get("Location"))
	// Output:
	// Status code: 410
	// Location:

}

func ExampleNewPingHandler_correct() {
	stor := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handler := handlers.NewPingHandler(stor, logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)

	// Output:
	// Status code: 200

}

func ExampleNewCreateShortURLJSON_correct() {
	mock := &handlers.MockStorage{Data: make(map[string]handlers.URLData)}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Создаем хендлер
	handler := handlers.NewCreateShortURLJSON(mock, "http://localhost", logger.Sugar())

	// Создаем запрос
	reqBody := `{"url": "http://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Подставляем userID в контекст
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)

	// Создаём ResponseRecorder
	rec := httptest.NewRecorder()

	// Вызываем хендлер
	handler.ServeHTTP(rec, req)

	// Печатаем результат
	fmt.Println("Status code:", rec.Code)
	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))

	// Output:
	// Status code: 201
	// Content-Type: application/json
}

func ExampleNewCreateShortURLJSON_empty_json() {
	mock := &handlers.MockStorage{Data: make(map[string]handlers.URLData)}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Создаем хендлер
	handler := handlers.NewCreateShortURLJSON(mock, "http://localhost", logger.Sugar())

	// Создаем запрос
	reqBody := `{}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Подставляем userID в контекст
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)

	// Создаём ResponseRecorder
	rec := httptest.NewRecorder()

	// Вызываем хендлер
	handler.ServeHTTP(rec, req)

	// Печатаем результат
	fmt.Println("Status code:", rec.Code)
	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))

	// Output:
	// Status code: 400
	// Content-Type: text/plain; charset=utf-8
}

func ExampleNewCreateBatchJSON() {
	mock := &handlers.MockStorage{Data: make(map[string]handlers.URLData)}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handler := handlers.NewCreateBatchJSON(mock, "http://localhost", logger.Sugar())

	// Формируем тело запроса
	reqBody := `[
		{"correlation_id": "1", "original_url": "http://example.com"},
		{"correlation_id": "2", "original_url": "http://example.org"}
	]`

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)
	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	body, _ := io.ReadAll(rec.Body)
	fmt.Println("Body:", string(body))

	// Output:
	// Status code: 201
	// Content-Type: application/json
	// Body: [{"correlation_id":"1","short_url":"http://localhost/mock123"},{"correlation_id":"2","short_url":"http://localhost/mock123"}]
}

func ExampleGetUserURLS() {
	// Создаём моковое хранилище с заранее заданными данными
	mock := &handlers.MockStorage{
		Data: map[string]handlers.URLData{
			"abc123": {OriginalURL: "http://example.com", UserID: "user123"},
			"xyz789": {OriginalURL: "http://golang.org", UserID: "user123"},
			"other":  {OriginalURL: "http://notincluded.com", UserID: "otherUser"},
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handler := handlers.GetUserURLS(mock, "http://localhost", logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user123")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)
	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	body, _ := io.ReadAll(rec.Body)
	fmt.Println("Body:", string(body))

	// Output:
	// Status code: 200
	// Content-Type: application/json
	// Body: [{"short_url":"http://localhost/abc123","original_url":"http://example.com"},{"short_url":"http://localhost/xyz789","original_url":"http://golang.org"}]
}
