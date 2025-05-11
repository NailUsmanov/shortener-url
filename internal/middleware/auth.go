package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

var secretKey []byte

func InitAuthMiddleWare(cfg *config.Config) {
	if len(cfg.CookieSecretKey) == 0 {
		panic("CookieSecretKey is empty")
	}
	secretKey = cfg.CookieSecretKey
}

func AuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Достаем куку из запроса
		cookie, err := r.Cookie("user_id")
		// Если ошибка или кука невалидна, создаем новое Айди пользователя и устанавливаем куку
		if err != nil || !isValidCookie(cookie) {
			UserID := generateUserID()
			setUserCookie(w, UserID)
			// Добавляем userID в контекст
			ctx := context.WithValue(r.Context(), UserIDKey, UserID)
			r = r.WithContext(ctx)
		} else {
			// Если кука валидна, извлекаем userID
			parts := strings.Split(cookie.Value, "|")
			ctx := context.WithValue(r.Context(), UserIDKey, parts[0])
			r = r.WithContext(ctx)
		}
		// Идем на следующий хендлер
		next.ServeHTTP(w, r)
	})
}

// Проверяет подпись куки.
func isValidCookie(cookie *http.Cookie) bool {
	// кука должна быть не нулевая
	if cookie == nil {
		return false
	}
	// разбиваем на две части, так как первая это UserID, а вторая часть это сама подпись
	parts := strings.Split(cookie.Value, "|")

	if len(parts) != 2 {
		return false
	}
	// Делаем подпись для конкретного юзера и сравниваем то, что получилось с тем, что передано сразу
	signature := generateSignature(parts[0])
	return signature == parts[1]
}

// Генерирует подпись для данных.
func generateSignature(data string) string {
	//создаем hmac
	h := hmac.New(sha256.New, secretKey)
	// добавляем данные data, переведенные из строки в байты, в HMAC объект
	h.Write([]byte(data + "/"))
	// h.Sum вычисляет хэш и записывает его в дст
	dst := h.Sum(nil)
	return hex.EncodeToString(dst)
}

// Генерирует ID пользователя
func generateUserID() string {
	return "user" + uuid.New().String()
}

// Устанавливает подписанную куку.
func setUserCookie(w http.ResponseWriter, UserID string) {
	data := generateSignature(UserID)
	cookieValue := UserID + "|" + data
	http.SetCookie(w, &http.Cookie{
		Name:   "user_id",
		Value:  cookieValue,
		Path:   "/",
		MaxAge: 30 * 24 * 60,
	})
}
