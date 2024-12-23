package main

import (
	"net/http"
	"time"
)

type RequestStats struct {
	statusCode   int
	started      time.Time
	bytesWritten int
	inner        http.ResponseWriter
}

func NewRequestStats(inner http.ResponseWriter) *RequestStats {
	return &RequestStats{
		statusCode: 200,
		started:    time.Now(),
		inner:      inner,
	}
}

func (r *RequestStats) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.inner.WriteHeader(statusCode)
}

func (r *RequestStats) Write(b []byte) (int, error) {
	n, err := r.inner.Write(b)
	r.bytesWritten += n
	return n, err
}
