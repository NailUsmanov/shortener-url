package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/internal/tasks"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type URLData struct {
	OriginalURL string
	UserID      string
}

type MockStorage struct {
	Data map[string]URLData
}

func (m *MockStorage) Save(ctx context.Context, url string, userID string) (string, error) {
	key := "mock123"
	m.Data[key] = URLData{
		OriginalURL: url,
		UserID:      userID,
	}
	return key, nil
}

func (m *MockStorage) Get(ctx context.Context, key string) (string, error) {
	if url, exists := m.Data[key]; exists {
		return url.OriginalURL, nil
	}
	return "", storage.ErrNotFound
}

func (m *MockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MockStorage) SaveInBatch(ctx context.Context, urls []string, userID string) ([]string, error) {
	keys := make([]string, len(urls))
	for i := range urls {
		keys[i] = "mock123" // Генерируем уникальные ключи
	}
	return keys, nil
}

func (m *MockStorage) GetByURL(ctx context.Context, originalURL string, userID string) (string, error) {
	return "", nil
}

func (m *MockStorage) GetUserURLS(ctx context.Context, userID string) (map[string]string, error) {
	result := make(map[string]string)

	for short, data := range m.Data {
		if data.UserID == userID {
			result[short] = data.OriginalURL
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (m *MockStorage) MarkAsDeleted(ctx context.Context, urls []string, userID string) error {
	for _, shortURL := range urls {
		if _, exists := m.Data[shortURL]; !exists {
			return fmt.Errorf("shortURL %s not found", shortURL)
		}
	}
	return nil
}

func (m *MockStorage) CountURL(ctx context.Context) (int, error) {
	return len(m.Data), nil
}

func (m *MockStorage) CountUsers(ctx context.Context) (int, error) {
	userSet := make(map[string]struct{})
	for _, user := range m.Data {
		userSet[user.UserID] = struct{}{}
	}
	return len(userSet), nil
}

func TestCreateShortURL(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		wantBody    string
		checkID     bool
	}{
		{
			name:        "Valid URL",
			requestBody: "http://test.ru/testcase12345",
			wantStatus:  http.StatusCreated,
			wantBody:    "http://test/mock123",
			checkID:     false,
		},
		{
			name:        "Empty body",
			requestBody: "",
			wantStatus:  http.StatusBadRequest,
			wantBody:    "Invalid request body\n",
			checkID:     false,
		},
		{
			name:        "Very short URL",
			requestBody: "http://t.ru",
			wantStatus:  http.StatusCreated,
			wantBody:    "http://test/mock123",
			checkID:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{Data: make(map[string]URLData)}
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
				s.Data["abc123"] = URLData{
					OriginalURL: "http://test.com",
					UserID:      "1",
				}
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
			storage := &MockStorage{Data: make(map[string]URLData)}
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
			storage := &MockStorage{Data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))

			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test_user")
			req = req.WithContext(ctx)

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
			storage := &MockStorage{Data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()
			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем user_id в контекст
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test_user")
			req = req.WithContext(ctx)

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
                {"correlation_id": "1", "short_url": "http://test/mock123"},
                {"correlation_id": "2", "short_url": "http://test/mock123"}
            ]`,
		},
		{
			name:        "Invalid JSON requesttest",
			requestBody: `[ { invalid json } ]`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"error":"Invalid JSON format"}`,
		},
		{
			name:        "Empty array",
			requestBody: `[]`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"error":"Empty batch request"}`,
		},
		{
			name:        "Missing Content-Type",
			requestBody: `[{"correlation_id": "1", "original_url": "http://test.com"}]`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"error":"Content-Type must be application/json"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{Data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewCreateBatchJSON(storage, "http://test", logger.Sugar())
			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.requestBody))
			if tt.name != "Missing Content-Type" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Добавляем user_id в контекст
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test_user")
			req = req.WithContext(ctx)

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

func TestGetUserURLS(t *testing.T) {
	// Создаем тестовое хранилище
	storage := storage.NewMemoryStorage()
	baseURL := "http://test"
	logger := zap.NewNop()

	// Тестовые данные
	userID := "user1"
	testURLs := map[string]string{
		"abc": "http://example.com/1",
		"def": "http://example.com/2",
	}

	// Сохраняем тестовые URL
	ctx := context.Background()
	for _, original := range testURLs {
		_, err := storage.Save(ctx, original, userID)
		require.NoError(t, err)
	}

	t.Run("Успешный возврат URL пользователя", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))

		w := httptest.NewRecorder()
		GetUserURLS(storage, baseURL, logger.Sugar())(w, req)

		res := w.Result()
		defer res.Body.Close()

		// Проверяем статус код
		require.Equal(t, http.StatusOK, res.StatusCode)

		// Проверяем заголовок Content-Type
		require.Equal(t, "application/json", res.Header.Get("Content-Type"))

		// Декодируем ответ
		var response []struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		}
		err := json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)

		// Проверяем количество URL в ответе
		require.Len(t, response, len(testURLs))
	})

	t.Run("Нет URL для пользователя", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "unknown_user"))

		w := httptest.NewRecorder()
		GetUserURLS(storage, baseURL, logger.Sugar())(w, req)

		res := w.Result()
		defer res.Body.Close()

		require.Equal(t, http.StatusNoContent, res.StatusCode)
	})

	t.Run("Неавторизованный доступ", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)

		w := httptest.NewRecorder()
		GetUserURLS(storage, baseURL, logger.Sugar())(w, req)

		res := w.Result()
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}

func TestDeleteHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	defer logger.Sync()

	mockStore := &MockStorage{Data: map[string]URLData{
		"abc123": {OriginalURL: "http://example.com", UserID: "test-user"},
		"def456": {OriginalURL: "http://test.com", UserID: "test-user"},
	}}

	ch := make(chan tasks.DeleteTask, 1)

	handler := DeleteHandler(mockStore, sugar, ch)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "test-user")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Delete("/api/user/urls", handler)

	t.Run("valid request", func(t *testing.T) {
		payload, _ := json.Marshal([]string{"abc123", "def456"})
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusAccepted, rr.Code)

		select {
		case task := <-ch:
			require.Equal(t, "test-user", task.UserID)
			require.ElementsMatch(t, []string{"abc123", "def456"}, task.ShortURLs)
		default:
			t.Fatal("task not sent to channel")
		}
	})

	t.Run("missing content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader([]byte(`["abc123"]`)))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("no user id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader([]byte(`["abc123"]`)))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		DeleteHandler(mockStore, sugar, ch).ServeHTTP(rr, req)

		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGetStats(t *testing.T) {
	logger := zap.NewNop()
	sugar := logger.Sugar()

	// Создаём моковое хранилище и добавляем данные
	mockStore := &MockStorage{Data: map[string]URLData{
		"short1": {OriginalURL: "http://example.com/1", UserID: "user1"},
		"short2": {OriginalURL: "http://example.com/2", UserID: "user2"},
	}}

	// Разрешённая подсеть 192.168.0.0/16
	_, subnet, err := net.ParseCIDR("192.168.0.0/16")
	require.NoError(t, err)

	handler := GetStats(mockStore, subnet, sugar)

	t.Run("valid IP in subnet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.10")

		w := httptest.NewRecorder()
		handler(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var stats struct {
			URLs  int `json:"urls"`
			Users int `json:"users"`
		}
		err := json.NewDecoder(res.Body).Decode(&stats)
		require.NoError(t, err)

		assert.Equal(t, 2, stats.URLs)
		assert.Equal(t, 2, stats.Users)
	})

	t.Run("IP not in subnet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1") // Вне подсети

		w := httptest.NewRecorder()
		handler(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusForbidden, res.StatusCode)
	})
}
