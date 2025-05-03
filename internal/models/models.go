package models

type RequestURL struct {
	URL string `json:"url"`
}

type Response struct {
	Result string `json:"result"`
}

type RequestURLMassiv struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ResponseMassiv struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
