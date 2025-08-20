// Package handlers описывает функции обработчики, используемые в HTTP-запросах.
package handlers

import (
	_ "bytes"
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/models"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"go.uber.org/zap"
)

// NewPingHandler проверяет работоспособность функции обработчика.
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

// GetUserURLS выдает все существующие у пользователя короткие URL.
func GetUserURLS(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok || userID == "" {
			w.WriteHeader(http.StatusUnauthorized) // 401 для неавторизованных
			return
		}

		urls, err := s.GetUserURLS(r.Context(), userID)
		if err != nil {
			sugar.Errorf("GetUserURLS error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		keys := make([]string, 0, len(urls))
		for k := range urls {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var resp []models.UserURLs
		for _, short := range keys {
			resp = append(resp, models.UserURLs{
				ShortURL:    baseURL + "/" + short,
				OriginalURL: urls[short],
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

// GetStats выдает количество сокращенных URL в сервисе и количество пользователей в сервисе.
func GetStats(s storage.Storage, subnet *net.IPNet, sugar *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// смотрим заголовок запроса X-Real-IP
		ipStr := r.Header.Get("X-Real-IP")

		// парсим ip
		ip := net.ParseIP(ipStr)

		if ip == nil {
			// если заголовок X-Real-IP пуст, пробуем X-Forwarded-For
			// этот заголовок содержит адреса отправителя и промежуточных прокси
			// в виде 203.0.113.195, 70.41.3.18, 150.172.238.178
			ips := r.Header.Get("X-Forwarded-For")
			// разделяем цепочку адресов
			ipStrs := strings.Split(ips, ",")
			// интересует только первый
			ipStr = ipStrs[0]
			// парсим
			ip = net.ParseIP(ipStr)
		}

		if ip == nil {
			w.WriteHeader(http.StatusForbidden)
			sugar.Error("failed parse ip from http header")
			return
		}

		if !subnet.Contains(ip) {
			w.WriteHeader(http.StatusForbidden)
			sugar.Warnf("unathorized ip access")
			return
		}

		totalUser, err := s.CountUsers(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		totalURLs, err := s.CountURL(r.Context())

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		stats := models.StatsURLs{
			URLs:  totalURLs,
			Users: totalUser,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(stats); err != nil {
			sugar.Error("error encoding stats response:", err)
		}
	}
}
