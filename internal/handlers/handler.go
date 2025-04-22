package handlers

import (
	_ "bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NailUsmanov/practicum-shortener-url/internal/models"
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

// POST
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

// GET
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

// POST JSON api/shorten
func (h *URLHandler) CreateShortURLJSON(w http.ResponseWriter, r *http.Request) {

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		h.sugar.Error("Invalid content type:", contentType)
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
		return
	}

	var req models.RequestURL
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sugar.Error("cannot decode request JSON body:", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if len(req.URL) == 0 {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	//Валидность URL
	_, err := url.ParseRequestURI(req.URL)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	key, err := h.storage.Save(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resp models.Response
	resp.Result = h.baseURL + "/" + key

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		h.sugar.Error("error encoding response")
	}

}

func GzipMiddleware(h http.Handler, logger *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportGzip := strings.Contains(acceptEncoding, "gzip")
		contentType := r.Header.Get("Content-Type")
		supportTypeJSON := strings.Contains(contentType, "application/json")
		supportTypeHTML := strings.Contains(contentType, "text/html")
		if supportGzip && (supportTypeHTML || supportTypeJSON) {
			cw := NewCompressWriter(w)
			ow = cw
			defer cw.Close()
			ow.Header().Set("Content-Encoding", "gzip")
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body, logger)
			if err != nil {
				if logger != nil {
					logger.Errorf("Failed to decompress request: %v", err)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}
		h.ServeHTTP(ow, r)
	})
}
