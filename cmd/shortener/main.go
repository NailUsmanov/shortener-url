package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
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
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	originalURL := string(body)
	shortID := generateShortCode(8)

	StorageURL[shortID] = originalURL

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "http://localhost:8080/%s", shortID)

}

func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusBadGateway)
		return
	}
	id := r.PathValue("id")

	longURL, exists := StorageURL[id]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)

}

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", postHandler)
	mux.HandleFunc("GET /{id}", getHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}

}
