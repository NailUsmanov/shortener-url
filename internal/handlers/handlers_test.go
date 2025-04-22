package handlers

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type MockStorage struct {
	data map[string]string
}

func (m *MockStorage) Save(url string) (string, error) {
	key := "mock123"
	m.data[key] = url
	return key, nil
}

func (m *MockStorage) Get(key string) (string, error) {
	if url, exists := m.data[key]; exists {
		return url, nil
	}
	return "", errors.New("URL not found")
}

func TestCreateShortURL(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "Valid URL",
			requestBody: "http://test.ru/testcase12345",
			wantStatus:  http.StatusCreated,
			wantBody:    "http://test/mock123",
		},
		{
			name:        "Empty body",
			requestBody: "",
			wantStatus:  http.StatusBadRequest,
			wantBody:    "Invalid request body\n",
		},
		{
			name:        "Very short URL",
			requestBody: "http://t.ru",
			wantStatus:  http.StatusCreated,
			wantBody:    "http://test/mock123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{data: make(map[string]string)}
			logger, err := zap.NewDevelopment()
			if err != nil {
				// вызываем панику, если ошибка
				panic(err)
			}
			defer logger.Sync()

			// делаем регистратор SugaredLogger
			sugar := logger.Sugar()

			handler := NewURLHandler(storage, "http://test", sugar)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()

			handler.CreateShortURL(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantStatus == http.StatusCreated {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				fullURL := string(body)
				parts := strings.Split(fullURL, "/")
				shortID := parts[len(parts)-1]

				// Проверка символов shortID
				for _, char := range shortID {
					assert.True(t, strings.ContainsRune(chars, char),
						"ShortID содержит недопустимый символ: %c", char)
				}

				assert.Equal(t, tt.wantBody, fullURL)
			} else {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, tt.wantBody, string(body))
			}
		})
	}
}

func TestURLHandler_Redirect(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(s *MockStorage)
		urlParam   string
		wantStatus int
		wantHeader string
	}{
		{
			name: "Valid short URL",
			setup: func(s *MockStorage) {
				s.data["abc123"] = "http://test.com"
			},
			urlParam:   "abc123",
			wantStatus: http.StatusTemporaryRedirect,
			wantHeader: "http://test.com",
		},
		{
			name:       "Non-existent short URL",
			setup:      func(s *MockStorage) {},
			urlParam:   "invalid",
			wantStatus: http.StatusNotFound,
			wantHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{data: make(map[string]string)}
			tt.setup(storage)

			logger, err := zap.NewDevelopment()
			if err != nil {
				// вызываем панику, если ошибка
				panic(err)
			}
			defer logger.Sync()

			// делаем регистратор SugaredLogger
			sugar := logger.Sugar()

			handler := NewURLHandler(storage, "http://test", sugar)

			router := chi.NewRouter()
			router.Get("/{id}", handler.Redirect)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.urlParam, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantHeader != "" {
				assert.Equal(t, tt.wantHeader, res.Header.Get("Location"))
			}
		})
	}
}

func TestCreateShortURLJSON(t *testing.T) {

	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		bodyJSON    string
	}{
		{
			name:        "Valid URL JSON",
			requestBody: `{"url":"http://test.ru/testcase12345"}`,
			wantStatus:  http.StatusCreated,
			bodyJSON:    `{"result":"http://test/mock123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{data: make(map[string]string)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewURLHandler(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))

			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler.CreateShortURLJSON(w, req)

			res := w.Result()
			defer res.Body.Close()

			if tt.wantStatus == http.StatusCreated {

				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.bodyJSON, string(body))

			} else {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, tt.bodyJSON, string(body))
			}

		})
	}

}

func TestCreateShortURLJSONErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
	}{
		{
			name:        "Invalid JSON",
			requestBody: `{"url": "no closing quote}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Empty JSON body",
			requestBody: `{}`,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{data: make(map[string]string)}
			handler := NewURLHandler(storage, "http://test", zap.NewNop().Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateShortURLJSON(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestGzipMiddleWare(t *testing.T) {
	mockMiddleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	handler := GzipMiddleware(mockMiddleware, nil)
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
}

func TestGzipMiddlewareErrorCases(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Invalid gzip request", func(t *testing.T) {
		var buf bytes.Buffer
		// Пишем невалидные gzip данные
		buf.Write([]byte("invalid gzip data"))

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()

		handler := GzipMiddleware(mockHandler, zap.NewNop().Sugar())
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	})
}
