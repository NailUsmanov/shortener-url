package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAppRoutes(t *testing.T) {
	// Инициализация
	mockStore := storage.NewMemoryStorage()
	logger := zap.NewNop()
	defer logger.Sync()
	var subnet *net.IPNet
	app := NewApp(mockStore, "http://test", logger.Sugar(), subnet)

	t.Run("Create and redirect URL", func(t *testing.T) {
		// Шаг 1: Создаем короткую ссылку
		reqBody := strings.NewReader("https://example.com")
		req := newTestRequest(t, "POST", "/", reqBody)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		app.router.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Проверяем успешное создание
		assert.Equal(t, http.StatusCreated, res.StatusCode, "Expected 201 status code")

		// Шаг 2: Получаем short URL из ответа
		shortURL := strings.TrimSpace(readBody(t, res))
		require.True(t, strings.HasPrefix(shortURL, "http://test/"), "Invalid short URL format")

		// Шаг 3: Проверяем редирект
		key := strings.TrimPrefix(shortURL, "http://test/")
		reqRedirect := newTestRequest(t, "GET", "/"+key, nil)

		recRedirect := httptest.NewRecorder()
		app.router.ServeHTTP(recRedirect, reqRedirect)

		resRedirect := recRedirect.Result()
		defer resRedirect.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resRedirect.StatusCode)
		assert.Equal(t, "https://example.com", resRedirect.Header.Get("Location"))
	})
}

// Вспомогательная функция для создания запросов
func newTestRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, path, body)
	require.NoError(t, err, "Failed to create request")
	return req
}

// Вспомогательная функция для чтения тела ответа
func readBody(t *testing.T, res *http.Response) string {
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Failed to read response body")
	return string(body)
}
