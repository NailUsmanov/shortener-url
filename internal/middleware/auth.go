package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// Ключ для хранения userID в контексте
type contextKey string

const (
	UserIDKey contextKey = "userID"
)

// AuthMiddleware - HTTP middleware, проводит аутентификацию пользователя.
//
// Использует куки. Если кука отстутствует, то генерируется новый токен и UserID.
// Если кука есть, то используется ее значение как UserID
// Затем она добавляется в контекст.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем куку auth_token (как требуют тесты Практикума)
		cookie, err := r.Cookie("auth_token")
		var userID string

		if err != nil {
			// 2. Если куки нет - генерируем новый токен и userID
			userID = generateUserID()
			setAuthCookie(w, userID)
		} else {
			// 3. Если кука есть - используем её значение как userID
			userID = cookie.Value
		}

		// 4. Добавляем userID в контекст
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Генерация нового userID
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Установка куки аутентификации
func setAuthCookie(w http.ResponseWriter, userID string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "auth_token", // именно такое имя требует Практикум
		Value: userID,
		Path:  "/",
		// Secure: true, // раскомментировать для HTTPS
		// HttpOnly: true, // защита от XSS
	})
}

// GetUserIDFromContext извлекает userID из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
