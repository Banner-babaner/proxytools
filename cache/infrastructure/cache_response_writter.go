package infrastructure

import (
	"bytes"
	"net/http"
)

type CacheResponseWriter struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func NewCacheResponseWriter(w http.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		buffer:         new(bytes.Buffer),
		statusCode:     http.StatusOK,
	}
}

func (cw *CacheResponseWriter) Write(data []byte) (int, error) {
	cw.buffer.Write(data)
	return cw.ResponseWriter.Write(data)
}

func (cw *CacheResponseWriter) WriteHeader(statusCode int) {
	cw.statusCode = statusCode
	cw.ResponseWriter.WriteHeader(statusCode)
}

func (cw *CacheResponseWriter) Body() []byte {
	return cw.buffer.Bytes()
}

func (cw *CacheResponseWriter) StatusCode() int {
	return cw.statusCode
}