package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"github.com/go-chi/chi"
)

var StorageURL = make(map[string]string)

func generateShortCode(lenght int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, lenght)

	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusBadGateway)
		return
	}
	id := chi.URLParam(r, "id")

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
	r.Route("/", func(r chi.Router) {
		r.Post("/", postHandler)
		r.Get("/{id}", getHandler)
	})
	fmt.Printf("Server working on a port%s", config.FlagRunAddr)
	err := http.ListenAndServe(config.FlagRunAddr, r)
	if err != nil {
		panic(err)
	}

}
