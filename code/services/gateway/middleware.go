package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var requestTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path"},
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP request duration",
	},
	[]string{"method", "path"},
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		requestTotal.WithLabelValues(r.Method, r.URL.Path).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path).
			Observe(time.Since(start).Seconds())
	})
}

func ConcurrencyLimitMiddleware(limit int, next http.HandlerFunc) http.HandlerFunc {
	if limit <= 0 {
		return next
	}

	sem := make(chan struct{}, limit)
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next(w, r)
		default:
			http.Error(w, "gateway is overloaded", http.StatusServiceUnavailable)
		}
	}
}

func RegisterMetrics() {
	prometheus.MustRegister(requestTotal)
	prometheus.MustRegister(requestDuration)
}
