package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ufm/internal/monitoring/metrics"
)

// MetricsMiddleware adds Prometheus metrics to HTTP requests
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment requests in flight
		metrics.HttpRequestsInFlight.Inc()
		defer metrics.HttpRequestsInFlight.Dec()

		// Process request
		c.Next()

		// Record metrics after request is processed
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Record request count
		metrics.HttpRequestsTotal.WithLabelValues(
			c.Request.Method,
			endpoint,
			statusCode,
		).Inc()

		// Record request duration
		metrics.HttpRequestDuration.WithLabelValues(
			c.Request.Method,
			endpoint,
		).Observe(duration)
	}
}
