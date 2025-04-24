package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestGzipMiddleWare(t *testing.T) {
	mockMiddleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	handler := GzipMiddleware(mockMiddleware)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	testBody := `{"input":"test"}`
	expectedResponse := `{"status":"ok"}`

	t.Run("compressed response", func(t *testing.T) {
		req, err := http.NewRequest("POST", srv.URL, bytes.NewBufferString(testBody))
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		defer zr.Close()

		body, err := io.ReadAll(zr)
		require.NoError(t, err)
		assert.JSONEq(t, expectedResponse, string(body))
	})

	t.Run("compressed request", func(t *testing.T) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write([]byte(testBody))
		zw.Close()

		req, err := http.NewRequest("POST", srv.URL, &buf)
		require.NoError(t, err)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, expectedResponse, string(body))
	})

	t.Run("decompresses gzipped request", func(t *testing.T) {
		// Создаем gzipped тело запроса
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write([]byte(`{"input":"test"}`))
		require.NoError(t, err)
		require.NoError(t, zw.Close())

		req := httptest.NewRequest("POST", "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "", res.Header.Get("Content-Encoding"))

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":"ok"}`, string(body))
	})

}

func TestGzipMiddlewareErrorCases(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called on gzip decompression error")
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Invalid gzip request", func(t *testing.T) {
		var buf bytes.Buffer
		// Пишем невалидные gzip данные
		buf.Write([]byte("invalid gzip data"))

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler := GzipMiddleware(mockHandler)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	})

}

func TestLoggingMiddleware(t *testing.T) {
	// Создаем тестовый логгер
	logger := zaptest.NewLogger(t)
	sugar := logger.Sugar()

	// Тестовый обработчик
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Создаем middleware с тестовым логгером
	handler := LoggingMiddleWare(sugar)(mockHandler)

	t.Run("logs request details", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		// Здесь можно добавить проверки логов, если используете zaptest
	})
}
