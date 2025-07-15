package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// CompressWriter — обёртка над http.ResponseWriter, выполняющая сжатие ответа в формате gzip.
type CompressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// NewCompressWriter создаёт новый CompressWriter с включённым gzip-сжатием.
func NewCompressWriter(w http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header возвращает заголовки HTTP-ответа.
func (c *CompressWriter) Header() http.Header {
	return c.w.Header()
}

// Write сжимает и записывает данные в тело HTTP-ответа.
func (c *CompressWriter) Write(p []byte) (int, error) {

	if c.Header().Get("Content-Encoding") == "" {
		c.Header().Set("Content-Encoding", "gzip")
	}
	return c.zw.Write(p)
}

// WriteHeader устанавливает статус-код HTTP-ответа и добавляет заголовок Content-Encoding.
func (c *CompressWriter) WriteHeader(statusCode int) {
	if c.Header().Get("Content-Encoding") == "" {
		c.Header().Set("Content-Encoding", "gzip")
	}
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *CompressWriter) Close() error {
	return c.zw.Close()
}

// CompressReader - обертка над http.Reader для чтения запроса, сжатого в формате gzip.
type CompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader создаёт новый CompressReader и инициализирует gzip.Reader.
func NewCompressReader(r io.ReadCloser) (*CompressReader, error) {

	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		r:  r,
		zr: zr,
	}, nil

}

// Read считывает и распаковывает данные из gzip-сжатого тела запроса.
func (c *CompressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

func (c *CompressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// GzipMiddleware — HTTP middleware, распаковывающее входящие gzip-запросы и сжимающее ответы, если клиент поддерживает gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка входящих данных
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := NewCompressReader(r.Body)
			if err != nil {
				http.Error(w, "failed to decompress gzip body", http.StatusBadRequest)
				return
			}
			defer cr.Close()
			r.Body = cr

			if r.Header.Get("Content-Type") == "application/x-gzip" {
				r.Header.Set("Content-Type", "text/plain")
			}
		}

		// Подготовка сжатия исходящих данных
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := NewCompressWriter(w)
			defer cw.Close()
			w = cw
		}

		next.ServeHTTP(w, r)
	})
}
