package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ufm/internal/log"
)

func TelemetryMiddleware(c *gin.Context) {
	startTime := time.Now()
	// For tracing add request ID
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		c.Header("X-Request-ID", requestID)
	}
	duration := time.Since(startTime)
	log.DefaultLogger.Infof("Telemetry API: %s %s %d %v [%s]",
		c.Request.Method,
		c.Request.URL.Path,
		c.Writer.Status(),
		duration,
		requestID,
	)

	// Add performance headers
	c.Header("X-API-Version", "1.0")
	c.Header("X-Service", "ufm-telemetry")

}
