package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPCollector exposes Prometheus metrics for inbound HTTP requests.
type HTTPCollector struct {
	registry        *prometheus.Registry
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
}

// NewHTTPCollector constructs a collector with default histograms/counters.
func NewHTTPCollector() (*HTTPCollector, error) {
	registry := prometheus.NewRegistry()

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "osintmcp",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "Latency distribution for inbound HTTP requests.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	requestTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "osintmcp",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of inbound HTTP requests.",
	}, []string{"method", "path", "status"})

	if err := registry.Register(requestDuration); err != nil {
		return nil, err
	}

	if err := registry.Register(requestTotal); err != nil {
		return nil, err
	}

	collector := &HTTPCollector{
		registry:        registry,
		requestDuration: requestDuration,
		requestTotal:    requestTotal,
	}

	return collector, nil
}

// Handler returns an HTTP handler for exposing Prometheus metrics.
func (c *HTTPCollector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

// InstrumentHandler wraps the provided handler to record HTTP metrics.
func (c *HTTPCollector) InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)
		path := r.URL.Path

		c.requestTotal.WithLabelValues(r.Method, path, status).Inc()
		c.requestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
