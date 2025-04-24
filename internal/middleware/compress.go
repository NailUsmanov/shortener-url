package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type CompressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewCompressWriter(w http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *CompressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *CompressWriter) Write(p []byte) (int, error) {

	if c.Header().Get("Content-Encoding") == "" {
		c.Header().Set("Content-Encoding", "gzip")
	}
	return c.zw.Write(p)
}

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

type CompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

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

func (c *CompressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

func (c *CompressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		contentType := r.Header.Get("Content-Type")
		isJSON := strings.Contains(contentType, "application/json")

		// Распаковка входящих данных
		if isJSON && r.Header.Get("Content-Encoding") == "gzip" {
			cr, err := NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = cr
			defer cr.Close()
		}

		// Подготовка сжатия исходящих данных
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if isJSON && acceptEncoding != "" && strings.Contains(acceptEncoding, "gzip") {
			cw := NewCompressWriter(w)
			defer cw.Close()
			w = cw
		}
		next.ServeHTTP(w, r)
	})
}
