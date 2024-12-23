package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var proxyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "proxy_duration_seconds",
	Help: "Duration of the probe request",
}, []string{"target", "code"})

var proxyCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "proxy_total",
	Help: "Total number of probes",
}, []string{"target", "code"})
var proxyResponceBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "proxy_responce_bytes",
	Help: "Total number of bytes sent in reposnce to probes",
}, []string{"target", "code"})

type Server struct {
	targetManager *Manager
}

func NewServer(ctx context.Context, configPath string) (*Server, error) {
	s := &Server{
		targetManager: NewManager(),
	}
	err := s.targetManager.ReloadConfig(ctx, configPath)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Proxy(w http.ResponseWriter, r *http.Request) {
	stats := NewRequestStats(w)
	target := r.URL.Query().Get("target")
	urlString := r.URL.Query().Get("url")

	defer func() {
		w.WriteHeader(stats.statusCode)
		elapsed := time.Since(stats.started)
		code := strconv.Itoa(stats.statusCode)
		proxyHistogram.WithLabelValues(target, code).Observe(elapsed.Seconds())
		proxyCounter.WithLabelValues(target, code).Inc()
		proxyResponceBytes.WithLabelValues(target, code).Add(float64(stats.bytesWritten))
	}()

	u, err := url.Parse(urlString)
	if err != nil {
		stats.Write([]byte(err.Error()))
		stats.WriteHeader(http.StatusBadRequest)
		return
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Path == "" {
		u.Path = "/metrics"
	}
	u.Host = target
	resp, err := http.Get(u.String())
	if err != nil {
		stats.Write([]byte(err.Error()))
		stats.WriteHeader(http.StatusInternalServerError)
		return
	}
	io.Copy(stats, resp.Body)
	w.WriteHeader(resp.StatusCode)
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) Report(w http.ResponseWriter, r *http.Request) {
}
