package handlers

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type URLHandler struct {
	storage storage.Storage //Зависимость через интерфейс
	baseURL string
	sugar   *zap.SugaredLogger
}

func NewURLHandler(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) *URLHandler {
	return &URLHandler{storage: s,
		baseURL: baseURL,
		sugar:   sugar,
	}
}

func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty reques body", http.StatusBadRequest)
		return
	}

	rawURL := string(body)

	//Валидность URL
	_, err = url.ParseRequestURI(rawURL)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	key, err := h.storage.Save(rawURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(h.baseURL + "/" + key))

}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "id")
	url, err := h.storage.Get(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Для метода ПОСТ
func WithLogging(h http.Handler, sugar *zap.SugaredLogger) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		// эндпоинт "/"
		uri := r.RequestURI

		method := r.Method

		// точка, где выполняется хендлер pingHandler
		h.ServeHTTP(w, r) // обслуживание оригинального запроса

		duration := time.Since(start)

		sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
		)

	}

	// Возвращаем расширенный хендлер
	return http.HandlerFunc(logFn)
}

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		size   int
		status int
	}

	// добавляем реализацию http.ResponseWriter

	logginigResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *logginigResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *logginigResponseWriter) WriteHeader(StatusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(StatusCode)
	r.responseData.status = StatusCode
}

// Для метода GET

func WithLoggingRedirect(h http.Handler, sugar *zap.SugaredLogger) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		responseData := &responseData{0, 0}

		lw := logginigResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		h.ServeHTTP(&lw, r)

		sugar.Infoln(
			"status", responseData.status,
			"size", responseData.size,
		)

	}
	return http.HandlerFunc(logFn)
}
