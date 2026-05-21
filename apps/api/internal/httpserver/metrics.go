package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "codeatlas_http_requests_total",
			Help: "Total HTTP requests by method, path pattern, and status code.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "codeatlas_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	ingestionJobsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "codeatlas_ingestion_jobs_active",
		Help: "Approximate count of repositories in non-ready ingestion states.",
	})
)

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		route := routeLabel(r.URL.Path)
		status := strconv.Itoa(rw.status)
		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// routeLabel collapses IDs for low-cardinality Prometheus labels.
func routeLabel(path string) string {
	if path == "" {
		return "/"
	}
	parts := splitPath(path)
	for i, p := range parts {
		if p == "" {
			continue
		}
		if isNumericID(p) {
			parts[i] = "{id}"
		}
	}
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += "/" + p
	}
	if out == "" {
		return "/"
	}
	return out
}

func splitPath(path string) []string {
	var parts []string
	cur := ""
	for _, c := range path {
		if c == '/' {
			parts = append(parts, cur)
			cur = ""
			continue
		}
		cur += string(c)
	}
	parts = append(parts, cur)
	return parts
}

func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
