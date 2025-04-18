package handlers

import (
	"compress/gzip"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {

	if c.Header().Get("Content-Encoding") == "" {
		c.Header().Set("Content-Encoding", "gzip")
	}
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.Header().Get("Content-Encoding") == "" {
		c.Header().Set("Content-Encoding", "gzip")
	}
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser, logger *zap.SugaredLogger) (*compressReader, error) {

	zr, err := gzip.NewReader(r)
	if err != nil {
		if logger != nil {
			logger.Errorf("Failed to create gzip reader: %v", err)
		}
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil

}

func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
