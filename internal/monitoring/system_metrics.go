package monitoring

import (
	"runtime"
	"time"

	"github.com/ufm/internal/monitoring/metrics"
)

// SystemMetricsCollector collects and updates system-level metrics
type SystemMetricsCollector struct {
	startTime time.Time
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector() *SystemMetricsCollector {
	return &SystemMetricsCollector{
		startTime: time.Now(),
	}
}

// UpdateMetrics updates all system metrics
func (smc *SystemMetricsCollector) UpdateMetrics() {
	// Update uptime
	uptime := time.Since(smc.startTime).Seconds()
	metrics.SystemUptime.Set(uptime)

	// Update goroutine count
	metrics.SystemGoroutines.Set(float64(runtime.NumGoroutine()))

	// Update memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics.SystemMemoryUsage.Set(float64(memStats.Alloc))
}

// StartPeriodicUpdates starts periodic metrics updates
func (smc *SystemMetricsCollector) StartPeriodicUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			smc.UpdateMetrics()
		}
	}()
}
