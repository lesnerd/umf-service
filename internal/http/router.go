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

	// API versioning
	apiV1 := engine.Group("/api/v1")

	// System routes (health checks, etc.)
	systemApi := apiV1.Group("/system")
	systemApi.GET("/ping", systemHandler.Ping)
	systemApi.GET("/health", systemHandler.Health)
	systemApi.GET("/readiness", systemHandler.Readiness)
	systemApi.GET("/version", systemHandler.Version)

	// Root level telemetry routes for convenience (optional)
	telemetryRoot := engine.Group("/telemetry")
	telemetryRoot.GET("/metrics/:switchId/:metricType", telemetryMiddlewareFunc, telemetryHandler.GetMetric)
	telemetryRoot.GET("/metrics/:switchId", telemetryMiddlewareFunc, telemetryHandler.ListMetrics)
	telemetryRoot.GET("/metrics", telemetryMiddlewareFunc, telemetryHandler.ListMetrics)
	// Additional observability endpoints
	telemetryRoot.GET("/performance", telemetryHandler.GetPerformanceMetrics)
	telemetryRoot.GET("/health", telemetryHandler.GetHealthStatus)
	telemetryRoot.GET("/switches", telemetryHandler.GetSwitchList)
	telemetryRoot.GET("/metric-types", telemetryHandler.GetMetricTypes)
}
