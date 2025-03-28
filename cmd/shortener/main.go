package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"github.com/go-chi/chi"
)

var StorageURL = make(map[string]string)

func generateShortCode(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	shortID := generateShortCode(8)
	StorageURL[shortID] = originalURL

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "%s/%s", config.BaseURL, shortID)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем ID любым возможным способом
	id := chi.URLParam(r, "id")
	if id == "" {
		// Парсим вручную для некорректных URL
		path := r.URL.Path
		if strings.Contains(path, "/") {
			parts := strings.Split(path, "/")
			if len(parts) > 1 {
				id = parts[len(parts)-1]
			}
		}
	}

	if id == "" {
		http.NotFound(w, r)
		return
	}

	longURL, exists := StorageURL[id]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	config.ParseFlag()
	r := chi.NewRouter()

	// Добавляем маршрут для обработки "битых" URL
	r.Get("/8080/{id}", getHandler)
	r.Get("/{id}", getHandler)
	r.Post("/", postHandler)

	http.ListenAndServe(":8080", r)
}
