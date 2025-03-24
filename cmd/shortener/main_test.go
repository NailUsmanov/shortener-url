package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func TestPostHandlerPositive(t *testing.T) {
	type want struct {
		CodeStatus     int
		ContentType    string
		LenghtShortID  int
		ShortIDCorrect bool
	}

	tests := []struct {
		name string
		URL  string
		want want
	}{
		{
			name: "simple",
			URL:  "http://test.ru/testcase12345",
			want: want{
				CodeStatus:     201,
				ContentType:    "text/plain",
				LenghtShortID:  8,
				ShortIDCorrect: true,
			},
		},
		{
			name: "short",
			URL:  "http://test.ru/t",
			want: want{
				CodeStatus:     201,
				ContentType:    "text/plain",
				LenghtShortID:  8,
				ShortIDCorrect: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.URL))
			w := httptest.NewRecorder()
			postHandler(w, request)

			result := w.Result()
			defer result.Body.Close()

			assert.Equal(t, test.want.CodeStatus, result.StatusCode)
			assert.Equal(t, test.want.ContentType, result.Header.Get("Content-Type"))

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			fullURL := string(body)

			prefix := "http://localhost:8080/"
			ShortID := strings.TrimPrefix(fullURL, prefix)
			arrChars := []rune(chars)

			for _, v := range ShortID {
				assert.True(t, slices.Contains(arrChars, v), "ShortID состоит из неправильных символов")
			}

			assert.Equal(t, test.want.LenghtShortID, len(ShortID), "Некорректная длина ShortID")

		})
	}
}

func TestPostHandlerNegative(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	w := httptest.NewRecorder()
	postHandler(w, request)

	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func TestGetHandler(t *testing.T) {
	StorageURL["abc123"] = "http://test.com"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{id}", getHandler)

	request := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, request)
	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, result.StatusCode)
	assert.Equal(t, "http://test.com", result.Header.Get("Location"))
}
