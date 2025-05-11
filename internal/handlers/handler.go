package handlers

import (
	_ "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/NailUsmanov/practicum-shortener-url/internal/models"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func NewCreateShortURL(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sugar.Infof("Request headers: %+v", r.Header)

		// Проверяем метод
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
			return
		}
		// Проверяем Content-Type
		// contentType := r.Header.Get("Content-Type")
		// if contentType != "" && !strings.HasPrefix(contentType, "text/plain") {
		// 	http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		// 	return
		// }
		sugar.Infof("Content-Type: %s", r.Header.Get("Content-Type"))
		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Проверяем чтобы тело было не 0
		if len(body) == 0 {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		sugar.Infof("Received request body: %q", body)
		rawURL := strings.TrimSpace(string(body))
		sugar.Infof("Received raw URL: %q", rawURL)

		// Проверяем валидность URL
		_, err = url.ParseRequestURI(rawURL)
		if err != nil {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		// Получаем userID из контекста
		userID, _ := r.Context().Value("user_id").(string)

		// Проверяем наличие оригинального УРЛ в нашей мапе
		existsKey, err := s.GetByURL(r.Context(), rawURL, userID)
		if err == nil && existsKey != "" {
			sugar.Infof("URL already exists: %s -> %s", rawURL, existsKey)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(baseURL + "/" + existsKey))
			return
		}

		// Обрабатываем другие ошибки (кроме "не найдено")
		if err != nil {
			sugar.Errorf("Error checking URL existence: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Сохраняем URL
		key, err := s.Save(r.Context(), rawURL, userID)
		if err != nil {
			sugar.Errorf("Failed to save URL: %v", err)
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		// Возвращаем ответ
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(baseURL + "/" + key)); err != nil {
			sugar.Errorf("Failed to write response: %v", err)
		}
	}
}

func NewRedirect(s storage.Storage, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем ID из URL
		key := chi.URLParam(r, "id")
		// 2. Ищем оригинальный URL
		url, err := s.Get(r.Context(), key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// 3. Делаем редирект
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func NewPingHandler(s storage.Storage, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			sugar.Errorf("Failed to open DataBase: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewCreateShortURLJSON(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Получаем UserID из контекста
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Проверяем метод
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
			return
		}

		var req models.RequestURL
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sugar.Error("cannot decode request JSON body:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if len(req.URL) == 0 {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Проверяем валидность URL
		_, err := url.ParseRequestURI(req.URL)
		if err != nil {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		// Сохраняем URL
		key, err := s.Save(r.Context(), req.URL, userID)
		if err != nil {
			if errors.Is(err, storage.ErrAlreadyHasKey) {
				var resp models.Response
				resp.Result = baseURL + "/" + key
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(resp)
				return
			}

			sugar.Errorf("Failed to save URL: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
			return
		}

		// Возвращаем ответ
		var resp models.Response
		resp.Result = baseURL + "/" + key
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			sugar.Error("error encoding response")
		}

	}
}

func NewCreateBatchJSON(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем UserID из контекста

		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Строгая проверка Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Content-Type must be application/json",
			})
			return
		}

		var req []models.RequestURLMassiv
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sugar.Error("cannot decode request JSON body:", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON format"})
			return
		}

		if len(req) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Empty batch request"})
			return
		}

		var urls []string
		for _, item := range req {
			if _, err := url.ParseRequestURI(item.OriginalURL); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Invalid URL: %s", item.OriginalURL)})
				return
			}
			urls = append(urls, item.OriginalURL)
		}

		keys, err := s.SaveInBatch(r.Context(), urls, userID)
		if err != nil {
			if errors.Is(err, storage.ErrAlreadyHasKey) {

				for _, url := range urls {
					if key, err := s.GetByURL(r.Context(), url, userID); err == nil {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusConflict)
						json.NewEncoder(w).Encode(map[string]string{
							"short_url": baseURL + "/" + key,
						})
						return
					}
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			sugar.Error("failed to save batch:", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		var resp []models.ResponseMassiv
		for i, key := range keys {
			resp = append(resp, models.ResponseMassiv{
				CorrelationID: req[i].CorrelationID,
				ShortURL:      baseURL + "/" + key,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			sugar.Error("error encoding response:", err)
		}
	}
}

// GET /api/user/urls
func GetUserURLS(s storage.Storage, baseURl string, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ID, ok := r.Context().Value("user_id").(string)
		if !ok || ID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		urls, err := s.GetUserURLS(r.Context(), ID)
		if err != nil {
			sugar.Errorf("GetUserURLS error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var resp []models.UserURLs
		for short, original := range urls {
			resp = append(resp, models.UserURLs{
				ShortURL:    baseURl + "/" + short,
				OriginalURL: original,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			sugar.Error("error encoding response:", err)
		}
	}
}
