// Package models описывает структуры запросов и ответов, используемых в эндпоинтах.
package models

// RequestURL содержит URL для сокращения.
type RequestURL struct {
	URL string `json:"url"`
}

// Response содержит сокращенный URL.
type Response struct {
	Result string `json:"result"`
}

// RequestURLMassiv содержит массив URL для дальнейшего сокращения.
type RequestURLMassiv struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// ResponseMassiv содержит ответ с уже сокращенными URL в массиве.
type ResponseMassiv struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLs содержит все пары сокращенного и оригинального URL конкретного пользователя.
type UserURLs struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// StatsURLs содержит количество всех сокращенных ссылок и количество всех пользователей.
type StatsURLs struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}
