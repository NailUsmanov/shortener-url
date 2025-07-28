package handlers

import (
	"errors"
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// NewRedirect перенаправляет клиента с короткой ссылки на оригинальный URL.
func NewRedirect(s storage.Storage, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем ID из URL
		key := chi.URLParam(r, "id")
		if key == "" {
			http.Error(w, "Empty URL ID", http.StatusBadRequest)
			return
		}
		// 2. Ищем оригинальный URL
		url, err := s.Get(r.Context(), key)
		if err != nil {
			sugar.Errorf("redirect error: %v", err)
		}
		switch {
		case errors.Is(err, storage.ErrDeleted):
			http.Error(w, "URL deleted", http.StatusGone)
			return
		case errors.Is(err, storage.ErrNotFound):
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		case err != nil:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// 3. Делаем редирект
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
