package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ufm/internal/log"
)

// MiddlewaresProvider provides tenant-related middleware
type MiddlewaresProvider interface {
	ExtractTenantIdGinMiddleware() gin.HandlerFunc
}

// multiTenantMiddlewaresProvider for multi-tenant deployments
type multiTenantMiddlewaresProvider struct{}

func NewMultiTenantMiddlewaresProvider() MiddlewaresProvider {
	return &multiTenantMiddlewaresProvider{}
}

func (p *multiTenantMiddlewaresProvider) ExtractTenantIdGinMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// For multi tenant, set a default tenant ID or skip
		c.Set("tenant_id", "default")
		c.Next()
	})
}

// HandleTraceIdSetupFunc sets up trace ID for request tracing
func HandleTraceIdSetupFunc(logger log.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Header("X-Trace-Id", traceID)
		c.Next()
	})
}

// HandleGinLogsFunc logs HTTP requests
func HandleGinLogsFunc(logger log.Logger, requestLogger log.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		traceID := ""
		if param.Keys != nil {
			if id, exists := param.Keys["trace_id"]; exists {
				traceID = id.(string)
			}
		}

		requestLogger.WithFields(map[string]interface{}{
			"trace_id":    traceID,
			"logger_name": "http-server",
			"method":      param.Method,
			"path":        param.Path,
			"status":      param.StatusCode,
			"latency":     param.Latency,
			"client_ip":   param.ClientIP,
			"user_agent":  param.Request.UserAgent(),
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
		}).Infof("%s %s %d %v", param.Method, param.Path, param.StatusCode, param.Latency)

		return ""
	})
}

// HandleUnexpectedPanicRecoveryFunc handles panics and recovers gracefully
func HandleUnexpectedPanicRecoveryFunc(logger log.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.WithFields(map[string]interface{}{
			"panic":       recovered,
			"logger_name": "http-server",
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"trace_id":    c.GetString("trace_id"),
		}).Errorf("Panic recovered: %v", recovered)

		c.JSON(500, gin.H{
			"error":   "Internal server error",
			"message": "An unexpected error occurred",
		})
	})
}
