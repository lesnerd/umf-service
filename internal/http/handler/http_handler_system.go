package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/service"
)

type SystemHandler interface {
	Ping(c *gin.Context)
	Health(c *gin.Context)
	Readiness(c *gin.Context)
	Version(c *gin.Context)
	Metrics(c *gin.Context)
}

type systemHandler struct {
	logger log.Logger
	ctx    service.Context
}

func NewSystemHandler(ctx service.Context) SystemHandler {
	return &systemHandler{
		logger: ctx.LoggerFactory().(log.LoggerFactory).GetLogger("system-handler"),
		ctx:    ctx,
	}
}

func (h *systemHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   service.PrettyName,
	})
}

func (h *systemHandler) Health(c *gin.Context) {
	// Perform health checks here (database, external services, etc.)
	healthy := true
	status := "healthy"
	statusCode := http.StatusOK

	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   service.PrettyName,
		"checks": gin.H{
			"database": "ok",
			// Add more health checks as needed
		},
	})
}

func (h *systemHandler) Readiness(c *gin.Context) {
	// Perform readiness checks here
	ready := true
	status := "ready"
	statusCode := http.StatusOK

	if !ready {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   service.PrettyName,
	})
}

func (h *systemHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": service.PrettyName,
		"version": "1.0.0",   // This should come from build info
		"build":   "dev",     // This should come from build info
		"commit":  "unknown", // This should come from build info
	})
}

func (h *systemHandler) Metrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
