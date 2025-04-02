package handlers

import (
	"io"
	"net/http"
	"net/url"

	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
)

type URLHandler struct {
	storage storage.Storage //Зависимость через интерфейс
	baseURL string
}

func NewURLHandler(s storage.Storage, baseURL string) *URLHandler {
	return &URLHandler{storage: s,
		baseURL: baseURL,
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
