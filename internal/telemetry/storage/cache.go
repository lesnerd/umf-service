package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/ufm/internal/telemetry/models"
)

// InMemoryCache implements TelemetryCache interface with thread-safe operations
type InMemoryCache struct {
	mu           sync.RWMutex
	data         map[string]*models.TelemetryData // switchID -> latest telemetry data
	lastUpdated  map[string]time.Time             // switchID -> last update time
	requestCount int64                            // Total requests served
	hitCount     int64                            // Cache hits
}

// NewInMemoryCache creates a new in-memory cache instance
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data:        make(map[string]*models.TelemetryData),
		lastUpdated: make(map[string]time.Time),
	}
}

// UpdateMetrics updates the metrics for a specific switch
func (c *InMemoryCache) UpdateMetrics(switchID string, data models.TelemetryData) error {
	if switchID == "" {
		return fmt.Errorf("switchID cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy of the data
	dataCopy := data
	dataCopy.SwitchID = switchID
	if dataCopy.Timestamp.IsZero() {
		dataCopy.Timestamp = time.Now()
	}

	c.data[switchID] = &dataCopy
	c.lastUpdated[switchID] = time.Now()

	return nil
}

// UpdateBatch updates metrics for multiple switches in a single operation
func (c *InMemoryCache) UpdateBatch(data map[string]models.TelemetryData) error {
	if len(data) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for switchID, telemetryData := range data {
		if switchID == "" {
			continue // Skip empty switch IDs
		}

		// Store a copy of the data
		dataCopy := telemetryData
		dataCopy.SwitchID = switchID
		if dataCopy.Timestamp.IsZero() {
			dataCopy.Timestamp = now
		}

		c.data[switchID] = &dataCopy
		c.lastUpdated[switchID] = now
	}

	return nil
}

// GetMetric retrieves a specific metric value for a switch
func (c *InMemoryCache) GetMetric(switchID string, metricType models.MetricType) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.requestCount++

	data, exists := c.data[switchID]
	if !exists {
		return nil, fmt.Errorf("switch %s not found", switchID)
	}

	c.hitCount++
	return data.GetMetricValue(metricType)
}

// GetAllMetrics retrieves all metrics for a specific switch
func (c *InMemoryCache) GetAllMetrics(switchID string) (*models.TelemetryData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.requestCount++

	data, exists := c.data[switchID]
	if !exists {
		return nil, fmt.Errorf("switch %s not found", switchID)
	}

	c.hitCount++

	// Return a copy to prevent external modifications
	dataCopy := *data
	return &dataCopy, nil
}

// ListAllSwitches returns a map of all switches and their latest metrics
func (c *InMemoryCache) ListAllSwitches() map[string]*models.TelemetryData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.requestCount++

	// Create a copy of the data to prevent external modifications
	result := make(map[string]*models.TelemetryData, len(c.data))
	for switchID, data := range c.data {
		dataCopy := *data
		result[switchID] = &dataCopy
	}

	if len(result) > 0 {
		c.hitCount++
	}

	return result
}

// GetSwitchCount returns the number of switches in the cache
func (c *InMemoryCache) GetSwitchCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.data)
}

// GetLastUpdate returns the last update time for a specific switch
func (c *InMemoryCache) GetLastUpdate(switchID string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if lastUpdate, exists := c.lastUpdated[switchID]; exists {
		return lastUpdate
	}
	return time.Time{} // Zero time if not found
}

// CleanupStale removes entries older than the specified max age
func (c *InMemoryCache) CleanupStale(maxAge time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removedCount := 0

	for switchID, lastUpdate := range c.lastUpdated {
		if lastUpdate.Before(cutoff) {
			delete(c.data, switchID)
			delete(c.lastUpdated, switchID)
			removedCount++
		}
	}

	return removedCount
}

// GetSnapshot returns a snapshot of all current data
func (c *InMemoryCache) GetSnapshot() *models.TelemetrySnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	snapshot := &models.TelemetrySnapshot{
		Timestamp:    now,
		GenerationID: fmt.Sprintf("cache_%d", now.UnixNano()),
		Switches:     make(map[string]*models.TelemetryData, len(c.data)),
	}

	// Create copies of all data
	for switchID, data := range c.data {
		dataCopy := *data
		snapshot.Switches[switchID] = &dataCopy
	}

	return snapshot
}

// GetCacheStats returns statistics about cache performance
func (c *InMemoryCache) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	if c.requestCount > 0 {
		hitRate = float64(c.hitCount) / float64(c.requestCount) * 100
	}

	return map[string]interface{}{
		"total_requests": c.requestCount,
		"cache_hits":     c.hitCount,
		"hit_rate":       fmt.Sprintf("%.2f%%", hitRate),
		"switch_count":   len(c.data),
		"memory_entries": len(c.data) + len(c.lastUpdated),
	}
}

// Clear removes all data from the cache
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*models.TelemetryData)
	c.lastUpdated = make(map[string]time.Time)
	c.requestCount = 0
	c.hitCount = 0
}

// GetMemoryUsage estimates the memory usage of the cache
func (c *InMemoryCache) GetMemoryUsage() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Rough estimation:
	// - Each TelemetryData struct: ~200 bytes
	// - Each map entry overhead: ~50 bytes
	// - String keys: ~20 bytes average
	// - Time values: ~24 bytes

	entrySize := int64(200 + 50 + 20 + 24) // ~294 bytes per entry
	return int64(len(c.data)) * entrySize
}
