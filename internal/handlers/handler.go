package handlers

import (
	_ "bytes"
	"encoding/json"
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
		contentType := r.Header.Get("Content-Type")
		if contentType != "" && !strings.HasPrefix(contentType, "text/plain") {
			http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
			return
		}
		sugar.Infof("Content-Type: %s", r.Header.Get("Content-Type"))
		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

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

		// Сохраняем URL
		key, err := s.Save(r.Context(), rawURL)
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
		key, err := s.Save(r.Context(), req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
			return
		}

		var req []models.RequestURLMassiv
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sugar.Error("cannot decode request JSON body:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if len(req) == 0 {
			http.Error(w, "Empty batch request", http.StatusBadRequest)
			return
		}

		var urls []string
		for _, item := range req {
			if _, err := url.ParseRequestURI(item.OriginalURL); err != nil {
				http.Error(w, fmt.Sprintf("Invalid URL: %s", item.OriginalURL), http.StatusBadRequest)
				return
			}
			urls = append(urls, item.OriginalURL)
		}
		var keys []string
		for i := range urls {
			key, err := s.Save(r.Context(), urls[i])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			keys = append(keys, key)
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
