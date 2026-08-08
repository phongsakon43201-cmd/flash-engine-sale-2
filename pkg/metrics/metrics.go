package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	once sync.Once

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	OrdersPlacedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flashsale_orders_placed_total",
			Help: "Total flash sale orders placed by status",
		},
		[]string{"status"},
	)

	RedisCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hits_total",
			Help: "Total Redis cache hit count",
		},
	)

	RedisCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_misses_total",
			Help: "Total Redis cache miss count",
		},
	)
)

// InitMetrics registers Prometheus metrics collectors once
func InitMetrics() {
	once.Do(func() {
		prometheus.MustRegister(
			HTTPRequestsTotal,
			HTTPRequestDuration,
			OrdersPlacedTotal,
			RedisCacheHits,
			RedisCacheMisses,
		)
	})
}

// PrometheusMiddleware collects latency and status metrics for Fiber HTTP endpoints
func PrometheusMiddleware() fiber.Handler {
	InitMetrics()
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Seconds()

		status := strconv.Itoa(c.Response().StatusCode())
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}

		HTTPRequestsTotal.WithLabelValues(c.Method(), path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Method(), path).Observe(duration)

		return err
	}
}

// PrometheusHandler exposes the /metrics endpoint
func PrometheusHandler() fiber.Handler {
	InitMetrics()
	return func(c *fiber.Ctx) error {
		handler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
		handler(c.Context())
		return nil
	}
}
