package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func (m *MockStorage) Save(ctx context.Context, url string) (string, error) {
	key := "mock123"
	m.data[key] = url
	return key, nil
}

func (m *MockStorage) Get(ctx context.Context, key string) (string, error) {
	if url, exists := m.data[key]; exists {
		return url, nil
	}
	return "", errors.New("URL not found")
}

func (m *MockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MockStorage) SaveInBatch(ctx context.Context, urls []string) ([]string, error) {
	keys := make([]string, len(urls))
	for i, url := range urls {
		key := "mock123_" + strconv.Itoa(i)
		m.data[key] = url
		keys[i] = key
	}
	return keys, nil
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

			handler := NewCreateShortURL(storage, "http://test", sugar)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()

			handler(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.Body == nil {
				t.Fatal("Response body is nil")
			}

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

			handler := NewRedirect(storage, sugar)

			router := chi.NewRouter()
			router.Get("/{id}", handler)

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

			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))

			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler(w, req)

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
			logger := zap.NewNop()

			defer logger.Sync()
			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestCreateBatchJSON(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		wantBody    string
	}{
		{
			name: "Valid test",
			requestBody: `[
                {"correlation_id": "1", "original_url": "http://test1.com"},
                {"correlation_id": "2", "original_url": "http://test2.com"}
            ]`,
			wantStatus: http.StatusCreated,
			wantBody: `[
                {"correlation_id": "1", "short_url": "http://test/mock123_0"},
                {"correlation_id": "2", "short_url": "http://test/mock123_1"}
            ]`,
		},
		{
			name:        "Invalid JSON requesttest",
			requestBody: `[ { invalid json } ]`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"error":"Invalid JSON format"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{data: make(map[string]string)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewCreateBatchJSON(storage, "http://test", logger.Sugar())
			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			if tt.wantStatus == http.StatusCreated {
				assert.JSONEq(t, tt.wantBody, string(body))
			} else {
				assert.JSONEq(t, tt.wantBody, string(body))
			}

		})
	}
}
