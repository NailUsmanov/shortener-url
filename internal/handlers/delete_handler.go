package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/internal/tasks"
	"go.uber.org/zap"
)

// DeleteHandler удаляет из памяти короткий URL.
func DeleteHandler(s storage.Storage, sugar *zap.SugaredLogger, ch chan tasks.DeleteTask) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Берем юзерИД из контекста
		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok || userID == "" {
			w.WriteHeader(http.StatusUnauthorized) // 401 для неавторизованных
			return
		}
		// Проверяем контент тайп
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid content type", http.StatusBadRequest)
			return
		}
		// Декодируем запрос
		var ShortURLs []string

		if err := json.NewDecoder(r.Body).Decode(&ShortURLs); err != nil {
			sugar.Error("cannot decode request JSON body:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}
		// Проверяем что массив не пустой
		if len(ShortURLs) == 0 {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		// Создаем ДелитТаск и отправляем в канал массив сокращенных урлов
		task := tasks.DeleteTask{
			UserID:    userID,
			ShortURLs: ShortURLs,
		}
		ch <- task

		// Выставляем статус Accepted
		w.WriteHeader(http.StatusAccepted)
	})
}
