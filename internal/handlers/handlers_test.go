package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type URLData struct {
	originalURL string
	userID      string
}

type MockStorage struct {
	data map[string]URLData
}

func (m *MockStorage) Save(ctx context.Context, url string, userID string) (string, error) {
	key := "mock123"
	m.data[key] = URLData{
		originalURL: url,
		userID:      userID,
	}
	return key, nil
}

func (m *MockStorage) Get(ctx context.Context, key string) (string, error) {
	if url, exists := m.data[key]; exists {
		return url.originalURL, nil
	}
	return "", errors.New("URL not found")
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

	for short, data := range m.data {
		if data.userID == userID {
			result[short] = data.originalURL
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
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
			storage := &MockStorage{data: make(map[string]URLData)}
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
				s.data["abc123"] = URLData{
					originalURL: "http://test.com",
					userID:      "1",
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
			storage := &MockStorage{data: make(map[string]URLData)}
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
			storage := &MockStorage{data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))

			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), "user_id", "test_user")
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
			storage := &MockStorage{data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()
			handler := NewCreateShortURLJSON(storage, "http://test", logger.Sugar())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем user_id в контекст
			ctx := context.WithValue(req.Context(), "user_id", "test_user")
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
			storage := &MockStorage{data: make(map[string]URLData)}
			logger := zap.NewNop()

			defer logger.Sync()

			handler := NewCreateBatchJSON(storage, "http://test", logger.Sugar())
			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.requestBody))
			if tt.name != "Missing Content-Type" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Добавляем user_id в контекст
			ctx := context.WithValue(req.Context(), "user_id", "test_user")
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
	type UserURLs struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}

	tests := []struct {
		name         string
		setupStorage func(s *MockStorage)
		contextValue interface{}
		wantStatus   int
		wantResponse []UserURLs
	}{
		{
			name: "Успешный возврат URL пользователя",
			setupStorage: func(s *MockStorage) {
				s.data = map[string]URLData{
					"abc123": {originalURL: "http://example.com/1", userID: "user1"},
					"def456": {originalURL: "http://example.com/2", userID: "user1"},
					"ghi789": {originalURL: "http://example.com/3", userID: "user2"},
				}
			},
			contextValue: "user1",
			wantStatus:   http.StatusOK,
			wantResponse: []UserURLs{
				{ShortURL: "http://test/abc123", OriginalURL: "http://example.com/1"},
				{ShortURL: "http://test/def456", OriginalURL: "http://example.com/2"},
			},
		},
		{
			name:         "Нет URL для пользователя",
			setupStorage: func(s *MockStorage) { s.data = make(map[string]URLData) },
			contextValue: "user1",
			wantStatus:   http.StatusNoContent,
			wantResponse: nil,
		},
		{
			name:         "Неавторизованный доступ (нет user_id в контексте)",
			setupStorage: func(s *MockStorage) {},
			contextValue: nil,
			wantStatus:   http.StatusUnauthorized,
			wantResponse: nil,
		},
		{
			name:         "Неавторизованный доступ (пустой user_id)",
			setupStorage: func(s *MockStorage) {},
			contextValue: "",
			wantStatus:   http.StatusUnauthorized,
			wantResponse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Инициализация мока хранилища
			storage := &MockStorage{data: make(map[string]URLData)}
			tt.setupStorage(storage)

			// Создаем логгер
			logger := zap.NewNop()
			defer logger.Sync()

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

			// Добавляем user_id в контекст если нужно
			if tt.contextValue != nil {
				ctx := context.WithValue(req.Context(), "user_id", tt.contextValue)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()

			// Вызываем хендлер напрямую
			GetUserURLS(storage, "http://test", logger.Sugar())(w, req)

			// Проверяем результат
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantResponse != nil {
				var response []UserURLs
				err := json.NewDecoder(res.Body).Decode(&response)
				require.NoError(t, err)

				// Сортируем для стабильного сравнения
				sort.Slice(response, func(i, j int) bool {
					return response[i].ShortURL < response[j].ShortURL
				})
				sort.Slice(tt.wantResponse, func(i, j int) bool {
					return tt.wantResponse[i].ShortURL < tt.wantResponse[j].ShortURL
				})

				assert.Equal(t, tt.wantResponse, response)
				assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
			} else if tt.wantStatus == http.StatusOK {
				// Проверяем что тело пустое, если не ожидаем ответа
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Empty(t, body)
			}
		})
	}
}
