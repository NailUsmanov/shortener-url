package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Введем все возможные символы для короткого URL
const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// // want описывает ожидаемые характеристики ответа:
//   - CodeStatus: HTTP-статус (например, 201 Created).
//   - ContentType: заголовок Content-Type (например, "text/plain").
//   - LenghtShortID: ожидаемая длина короткого идентификатора.
//   - ShortIDCorrect: должен ли идентификатор состоять из допустимых символов (chars).нны короткого URL, проверки корректен он или нет в соответствие с нашим chars
func TestPostHandlerPositive(t *testing.T) {
	type want struct {
		CodeStatus     int
		ContentType    string
		LengthShortID  int
		ShortIDCorrect bool
	}

	// tests определяет набор тестовых случаев:
	//   - name: название теста (для удобства отладки).
	//   - URL: исходный URL для сокращения.
	//   - want: ожидаемые результаты.

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
				LengthShortID:  8,
				ShortIDCorrect: true,
			},
		},
		{
			name: "short",
			URL:  "http://test.ru/t",
			want: want{
				CodeStatus:     201,
				ContentType:    "text/plain",
				LengthShortID:  8,
				ShortIDCorrect: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Создаем тестовый запрос в который передаем нашу ссылку для сокращения
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.URL))

			// Инициализируем тестовую запись ответа
			w := httptest.NewRecorder()

			// Запускаю обработчик с местом куда записывать и с запросом
			postHandler(w, request)

			result := w.Result()
			defer result.Body.Close()

			// Сравниваем желаемый статус код и то, что у нас выдается
			// Сравниваем желаемый тип с тем, что по факту
			assert.Equal(t, test.want.CodeStatus, result.StatusCode)
			assert.Equal(t, test.want.ContentType, result.Header.Get("Content-Type"))

			// Записываем данные из ответа сервера в body (передается в byte)
			// Поэтому потом переводим в string
			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			fullURL := string(body)

			// Проверяем, что ShortID состоит только из допустимых символов (chars).
			// Преобразуем chars в []rune для удобства проверки.
			// Отделяем префикс, чтобы потом от прочитанного URL можно было оставить
			// только наше сокращение
			parts := strings.Split(fullURL, "/")
			ShortID := parts[len(parts)-1]
			arrChars := []rune(chars)

			// Циклом проверяем каждую букву на соответствие
			for _, v := range ShortID {
				assert.True(t, slices.Contains(arrChars, v), "ShortID состоит из неправильных символов %s", ShortID)
			}

			assert.Equal(t, test.want.LengthShortID, len(ShortID), "Некорректная длина ShortID %s", ShortID)

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

	// Добавляем тестовую запись в мапу, исходный URL http://test.com, ключ abc123
	StorageURL["abc123"] = "http://test.com"

	// Создаем наш роутер и регистрируем обработчик getHandler для GET запроса по пути /{id}
	r := chi.NewRouter()
	r.Get("/{id}", getHandler)

	request := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, request)
	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, result.StatusCode)
	assert.Equal(t, "http://test.com", result.Header.Get("Location"))
}

func TestGetHandlerNegative(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/{id}", getHandler)

	request := httptest.NewRequest(http.MethodGet, "/invalid_id", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, request)
	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusNotFound, result.StatusCode)
}
