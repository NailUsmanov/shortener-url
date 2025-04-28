package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Для захвата данных ответа, которая иначе не доступна в мидлвеар
// получаем размер ответа в байт и статус код самого ответа
type responseData struct {
	size   int
	status int
}

// Обертка над стандартным http.ResponseWriter, чтобы
// перехватывать вызовы Write и WriteHeader, затем сохранять данные (размер и статус)
// в responseData и делегировать вызовы оригинальному ResponseWriter
type loggingResponseWritter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWritter) Write(p []byte) (int, error) {
	size, err := r.ResponseWriter.Write(p)
	r.responseData.size = size // здесь мы перехватываем размер ответа в байтах
	return size, err
}

func (r *loggingResponseWritter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // записываем статус
}

func LoggingMiddleWare(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
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
