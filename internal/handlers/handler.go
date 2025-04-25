package handlers

import (
	_ "bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

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
	return &URLHandler{
		storage: s,
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

// POST JSON api/shorten
func (h *URLHandler) CreateShortURLJSON(w http.ResponseWriter, r *http.Request) {

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") &&
		!strings.Contains(contentType, "application/x-gzip") {
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
