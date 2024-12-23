package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "requests_total",
	Help: "Total number of requests",
}, []string{"code", "method"})

func main() {

	configPath := flag.String("config", "config.yaml", "path to config file")
	port := flag.Int("port", 8001, "port to listen on")
	flag.Parse()

	ctx := context.Background()

	server, err := NewServer(ctx, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Heartbeat("/health"))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(h http.Handler) http.Handler {
		return promhttp.InstrumentHandlerCounter(requestsTotal, h)
	})

	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Get("/proxy", server.Proxy)
	r.Get("/api/v1/refresh", server.Refresh)
	r.Get("/api/v1/report", server.Report)
	r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", 400)
	}))

	http.ListenAndServe(fmt.Sprintf(":%d", *port), r)
}
