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
		// Распаковка входящих данных
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			// cr, err := NewCompressReader(r.Body)
			cr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}

			r.Body = cr
			defer cr.Close()

			if strings.Contains(r.Header.Get("Content-Type"), "application/x-gzip") {
				r.Header.Set("Content-Type", "application/json")
			}
		}

		// Подготовка сжатия исходящих данных

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			cw := NewCompressWriter(w)
			defer cw.Close()
			w = cw
		}
		next.ServeHTTP(w, r)
	})
}
