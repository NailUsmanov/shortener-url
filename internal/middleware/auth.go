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
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

var secretKey []byte

var sugar *zap.SugaredLogger

func InitAuthMiddleWare(cfg *config.Config, logger *zap.SugaredLogger) {
	if len(cfg.CookieSecretKey) == 0 {
		panic("CookieSecretKey is empty")
	}
	secretKey = cfg.CookieSecretKey
	sugar = logger
}

func AuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sugar == nil {
			panic("AuthMiddleWare: logger not initialized")
		}

		sugar.Infof("Processing %s %s", r.Method, r.URL.Path)

		// Для всех API endpoints устанавливаем тестового пользователя
		if strings.HasPrefix(r.URL.Path, "/api/") {
			sugar.Info("Bypassing auth for API endpoint")
			ctx := context.WithValue(r.Context(), UserIDKey, "test_user")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		sugar.Info("Processing regular endpoint")
		cookie, err := r.Cookie("user_id")
		if err != nil {
			sugar.Infof("No cookie found: %v", err)
		}

		if err != nil || !isValidCookie(cookie) {
			userID := generateUserID()
			sugar.Infof("Generated new user ID: %s", userID)
			setUserCookie(w, userID)
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		parts := strings.Split(cookie.Value, "|")
		sugar.Infof("Authenticated user: %s", parts[0])
		ctx := context.WithValue(r.Context(), UserIDKey, parts[0])
		next.ServeHTTP(w, r.WithContext(ctx))
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
