package http

import (
	"github.com/gin-gonic/gin"
	"github.com/ufm/internal/http/handler"
	"github.com/ufm/internal/http/middleware"
)

func RegisterHandlers(
	engine *gin.Engine,
	systemHandler handler.SystemHandler,
	telemetryHandler handler.TelemetryHandler,
) {
	telemetryMiddlewareFunc := middleware.TelemetryMiddleware
	metricsMiddlewareFunc := middleware.MetricsMiddleware

	// API versioning
	apiV1 := engine.Group("/api/v1")

	// System routes (health checks, etc.)
	systemApi := apiV1.Group("/system")
	systemApi.GET("/ping", metricsMiddlewareFunc(), systemHandler.Ping)
	systemApi.GET("/health", metricsMiddlewareFunc(), systemHandler.Health)
	systemApi.GET("/readiness", metricsMiddlewareFunc(), systemHandler.Readiness)
	systemApi.GET("/version", metricsMiddlewareFunc(), systemHandler.Version)
	systemApi.GET("/metrics", systemHandler.Metrics) // Prometheus metrics endpoint

	// Root level telemetry routes for convenience (optional)
	telemetryRoot := engine.Group("/telemetry")
	telemetryRoot.GET("/metrics/:switchId/:metricType", telemetryMiddlewareFunc, metricsMiddlewareFunc(), telemetryHandler.GetMetric)
	telemetryRoot.GET("/metrics/:switchId", telemetryMiddlewareFunc, metricsMiddlewareFunc(), telemetryHandler.ListMetrics)
	telemetryRoot.GET("/metrics", telemetryMiddlewareFunc, metricsMiddlewareFunc(), telemetryHandler.ListMetrics)
	// Additional observability endpoints
	telemetryRoot.GET("/performance", metricsMiddlewareFunc(), telemetryHandler.GetPerformanceMetrics)
	telemetryRoot.GET("/health", metricsMiddlewareFunc(), telemetryHandler.GetHealthStatus)
	telemetryRoot.GET("/switches", metricsMiddlewareFunc(), telemetryHandler.GetSwitchList)
	telemetryRoot.GET("/metric-types", metricsMiddlewareFunc(), telemetryHandler.GetMetricTypes)
}
