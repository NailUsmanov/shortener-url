// Package middleware содержит middleware-функции для HTTP-сервера.
//
// Включает логирование запросов, сжатие ответов и аутентификацию пользователей.
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseData содержит данные об ответе HTTP-сервера — размер в байтах и статус-код.
//
// Используется для захвата информации, которую нельзя получить напрямую из ResponseWriter.
type responseData struct {
	size   int
	status int
}

// Обертка над стандартным http.ResponseWriter, чтобы
// перехватывать вызовы Write и WriteHeader, затем сохранять данные (размер и статус)
// в responseData и делегировать вызовы оригинальному ResponseWriter.
type loggingResponseWritter struct {
	http.ResponseWriter
	responseData *responseData
}

// Write логирует и записывает данные в тело HTTP-ответа.
func (r *loggingResponseWritter) Write(p []byte) (int, error) {
	size, err := r.ResponseWriter.Write(p)
	r.responseData.size = size // здесь мы перехватываем размер ответа в байтах
	return size, err
}

// loggingResponseWritter.WriteHeader устанавливает статус код HTTP-ответа.
func (r *loggingResponseWritter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // записываем статус
}

// LoggingMiddleware возвращает middleware, логирующее HTTP-запросы.
//
// В лог записываются URI, метод, статус-код, размер ответа и длительность обработки.
func LoggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			responseData := responseData{0, 0}

			lw := loggingResponseWritter{
				ResponseWriter: w,
				responseData:   &responseData,
			}
			next.ServeHTTP(&lw, r)

			duration := time.Since(start)
			logger.Infoln(
				"uri", r.RequestURI,
				"method", r.Method,
				"status", responseData.status,
				"size", responseData.size,
				"duration", duration,
			)
		})
	}
}
