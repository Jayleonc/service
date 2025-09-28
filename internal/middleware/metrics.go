package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics collects HTTP request metrics for Prometheus.
func Metrics(registry *prometheus.Registry) gin.HandlerFunc {
	requests := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	if err := registry.Register(requests); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if hist, ok := existing.ExistingCollector.(*prometheus.HistogramVec); ok {
				requests = hist
			}
		} else {
			panic(err)
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		dur := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		requests.WithLabelValues(c.Request.Method, c.FullPath(), status).Observe(dur)
	}
}
