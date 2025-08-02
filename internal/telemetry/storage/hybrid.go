package storage

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry/models"
)

// TelemetryRepository defines the interface for persistent storage operations
type TelemetryRepository interface {
	// Switch operations
	CreateSwitch(ctx context.Context, sw models.Switch) error
	GetSwitch(ctx context.Context, switchID string) (*models.Switch, error)
	ListSwitches(ctx context.Context) ([]models.Switch, error)

	// Telemetry data operations
	StoreMetrics(ctx context.Context, metrics []models.TelemetryData) error
	GetLatestMetrics(ctx context.Context, switchID string) (*models.TelemetryData, error)
	GetHistoricalMetrics(ctx context.Context, switchID string, from, to time.Time) ([]models.TelemetryData, error)

	// Utility operations
	DeleteOldMetrics(ctx context.Context, olderThan time.Time) error
	GetMetricsCount(ctx context.Context) (int64, error)
}

// TelemetryCache defines the interface for in-memory cache operations
type TelemetryCache interface {
	// Write operations
	UpdateMetrics(switchID string, data models.TelemetryData) error
	UpdateBatch(data map[string]models.TelemetryData) error

	// Read operations
	GetMetric(switchID string, metricType models.MetricType) (interface{}, error)
	GetAllMetrics(switchID string) (*models.TelemetryData, error)
	ListAllSwitches() map[string]*models.TelemetryData

	// Utility operations
	GetSwitchCount() int
	GetLastUpdate(switchID string) time.Time
	CleanupStale(maxAge time.Duration) int
	GetSnapshot() *models.TelemetrySnapshot
}

// TelemetryStore defines the interface for the hybrid storage system
type TelemetryStore interface {
	// Combines cache and repository operations
	TelemetryCache

	// Persistence operations
	FlushToDatabase(ctx context.Context) error
	LoadFromDatabase(ctx context.Context) error
	StoreMetricsBulk(ctx context.Context, metrics []models.TelemetryData) error

	// Lifecycle operations
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Health and metrics
	GetPerformanceMetrics() *models.PerformanceMetrics
}

// HybridStoreConfig holds configuration for the hybrid storage
type HybridStoreConfig struct {
	FlushInterval time.Duration // How often to flush cache to database
	BatchSize     int           // Maximum batch size for database writes
	CacheTTL      time.Duration // How long to keep data in cache
	MaxRetries    int           // Maximum retry attempts for database operations
}

// DefaultHybridStoreConfig returns sensible defaults
func DefaultHybridStoreConfig() HybridStoreConfig {
	return HybridStoreConfig{
		FlushInterval: 30 * time.Second,
		BatchSize:     100,
		CacheTTL:      5 * time.Minute,
		MaxRetries:    3,
	}
}

// HybridStore implements TelemetryStore interface combining cache and persistence
type HybridStore struct {
	cache      TelemetryCache
	repository TelemetryRepository
	config     HybridStoreConfig
	logger     log.Logger

	// Background processing
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	flushQueue chan []models.TelemetryData

	// Performance metrics
	mu                 sync.RWMutex
	totalRequests      int64
	totalCacheHits     int64
	totalDBWrites      int64
	totalDBWriteErrors int64
	lastFlushTime      time.Time
	startTime          time.Time
	pendingFlushItems  int
}

// NewHybridStore creates a new hybrid storage instance
func NewHybridStore(
	cache TelemetryCache,
	repository TelemetryRepository,
	config HybridStoreConfig,
	logger log.Logger,
) *HybridStore {
	if logger == nil {
		logger = log.DefaultLogger
	}

	return &HybridStore{
		cache:      cache,
		repository: repository,
		config:     config,
		logger:     logger,
		flushQueue: make(chan []models.TelemetryData, 100), // Buffered channel
		startTime:  time.Now(),
	}
}

// Start initializes the hybrid store and starts background processes
func (hs *HybridStore) Start(ctx context.Context) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.ctx != nil {
		return fmt.Errorf("hybrid store already started")
	}

	hs.ctx, hs.cancel = context.WithCancel(ctx)
	hs.startTime = time.Now()

	// Start background flush worker
	hs.wg.Add(1)
	go hs.flushWorker()

	// Start periodic cache cleanup
	hs.wg.Add(1)
	go hs.cleanupWorker()

	// Start periodic metrics flush
	hs.wg.Add(1)
	go hs.periodicFlushWorker()

	hs.logger.Infof("Hybrid telemetry store started with flush interval: %v", hs.config.FlushInterval)
	return nil
}

// Stop gracefully shuts down the hybrid store
func (hs *HybridStore) Stop(ctx context.Context) error {
	hs.mu.Lock()
	if hs.cancel == nil {
		hs.mu.Unlock()
		return fmt.Errorf("hybrid store not started")
	}
	hs.logger.Infof("Stopping hybrid telemetry store (completing remaining processing)...")
	hs.cancel()
	hs.mu.Unlock()

	// Wait for background workers to finish with a shorter timeout
	done := make(chan struct{})
	go func() {
		hs.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		hs.logger.Infof("Hybrid telemetry store stopped gracefully")

		// Close database connection
		if repo, ok := hs.repository.(*PostgreSQLRepository); ok {
			if err := repo.Close(); err != nil {
				hs.logger.Warnf("Failed to close database connection: %v", err)
			} else {
				hs.logger.Infof("Database connection closed successfully")
			}
		}

		return nil
	case <-ctx.Done():
		hs.logger.Warnf("Hybrid telemetry store stop timed out: %v", ctx.Err())

		// Still try to close database connection even on timeout
		if repo, ok := hs.repository.(*PostgreSQLRepository); ok {
			if err := repo.Close(); err != nil {
				hs.logger.Warnf("Failed to close database connection: %v", err)
			}
		}

		return ctx.Err()
	}
}

// Cache interface implementations

// UpdateMetrics updates metrics in cache and queues for database write
func (hs *HybridStore) UpdateMetrics(switchID string, data models.TelemetryData) error {
	// Update cache immediately for fast reads
	err := hs.cache.UpdateMetrics(switchID, data)
	if err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}

	// Queue for database write (non-blocking)
	select {
	case hs.flushQueue <- []models.TelemetryData{data}:
		hs.mu.Lock()
		hs.pendingFlushItems++
		hs.mu.Unlock()
	default:
		// Queue is full, log warning but don't block
		hs.logger.Warnf("Flush queue is full, dropping telemetry data for switch %s", switchID)
	}

	hs.incrementRequestCount()
	return nil
}

// UpdateBatch updates multiple metrics in cache and queues for database write
func (hs *HybridStore) UpdateBatch(data map[string]models.TelemetryData) error {
	// Update cache immediately
	err := hs.cache.UpdateBatch(data)
	if err != nil {
		return fmt.Errorf("failed to update cache batch: %w", err)
	}

	// Convert to slice for database write
	var metrics []models.TelemetryData
	for switchID, telemetryData := range data {
		telemetryData.SwitchID = switchID
		metrics = append(metrics, telemetryData)
	}

	// Queue for database write (non-blocking)
	if len(metrics) > 0 {
		select {
		case hs.flushQueue <- metrics:
			hs.mu.Lock()
			hs.pendingFlushItems += len(metrics)
			hs.mu.Unlock()
		default:
			hs.logger.Warnf("Flush queue is full, dropping %d telemetry records", len(metrics))
		}
	}

	hs.incrementRequestCount()
	return nil
}

// StoreMetricsBulk stores all telemetry records without deduplication
func (hs *HybridStore) StoreMetricsBulk(ctx context.Context, metrics []models.TelemetryData) error {
	if len(metrics) == 0 {
		return nil
	}

	// Update cache with latest values (for real-time access)
	cacheData := make(map[string]models.TelemetryData)
	for _, metric := range metrics {
		cacheData[metric.SwitchID] = metric
	}

	err := hs.cache.UpdateBatch(cacheData)
	if err != nil {
		return fmt.Errorf("failed to update cache batch: %w", err)
	}

	// Store all records to database (for historical data)
	return hs.writeToDatabase(ctx, metrics)
}

// Read operations delegate to cache for speed
func (hs *HybridStore) GetMetric(switchID string, metricType models.MetricType) (interface{}, error) {
	hs.incrementRequestCount()
	hs.incrementCacheHit() // Assume cache hit since we only read from cache
	return hs.cache.GetMetric(switchID, metricType)
}

func (hs *HybridStore) GetAllMetrics(switchID string) (*models.TelemetryData, error) {
	hs.incrementRequestCount()
	hs.incrementCacheHit()
	return hs.cache.GetAllMetrics(switchID)
}

func (hs *HybridStore) ListAllSwitches() map[string]*models.TelemetryData {
	hs.incrementRequestCount()
	hs.incrementCacheHit()
	return hs.cache.ListAllSwitches()
}

func (hs *HybridStore) GetSwitchCount() int {
	return hs.cache.GetSwitchCount()
}

func (hs *HybridStore) GetLastUpdate(switchID string) time.Time {
	return hs.cache.GetLastUpdate(switchID)
}

func (hs *HybridStore) CleanupStale(maxAge time.Duration) int {
	return hs.cache.CleanupStale(maxAge)
}

func (hs *HybridStore) GetSnapshot() *models.TelemetrySnapshot {
	hs.incrementRequestCount()
	return hs.cache.GetSnapshot()
}

// Persistence operations

// FlushToDatabase manually triggers a flush of pending data
func (hs *HybridStore) FlushToDatabase(ctx context.Context) error {
	hs.logger.Debugf("Manual flush to database requested")

	// Collect all pending items from queue
	var allMetrics []models.TelemetryData

	// Drain the queue with timeout
	flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for {
		select {
		case metrics := <-hs.flushQueue:
			allMetrics = append(allMetrics, metrics...)
		case <-flushCtx.Done():
			goto flushData
		default:
			goto flushData
		}
	}

flushData:
	if len(allMetrics) == 0 {
		return nil
	}

	return hs.writeToDatabase(ctx, allMetrics)
}

// LoadFromDatabase loads recent data from database into cache
func (hs *HybridStore) LoadFromDatabase(ctx context.Context) error {
	// Get all switches first
	switches, err := hs.repository.ListSwitches(ctx)
	if err != nil {
		return fmt.Errorf("failed to load switches: %w", err)
	}

	loadedCount := 0
	for _, sw := range switches {
		// Get latest metrics for each switch
		metrics, err := hs.repository.GetLatestMetrics(ctx, sw.ID)
		if err != nil {
			hs.logger.Warnf("Failed to load latest metrics for switch %s: %v", sw.ID, err)
			continue
		}

		// Update cache
		err = hs.cache.UpdateMetrics(sw.ID, *metrics)
		if err != nil {
			hs.logger.Warnf("Failed to update cache for switch %s: %v", sw.ID, err)
			continue
		}

		loadedCount++
	}

	hs.logger.Infof("Loaded latest metrics for %d switches from database", loadedCount)
	return nil
}

// GetPerformanceMetrics returns comprehensive performance statistics
func (hs *HybridStore) GetPerformanceMetrics() *models.PerformanceMetrics {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate average age of data
	var avgDataAge float64
	switches := hs.cache.ListAllSwitches()
	if len(switches) > 0 {
		var totalAge time.Duration
		for _, data := range switches {
			age := time.Since(data.Timestamp)
			totalAge += age
		}
		avgDataAge = totalAge.Seconds() / float64(len(switches))
	}

	// Calculate API latency (rough estimate based on cache performance)
	apiLatency := 0.5 // Sub-millisecond for cache operations

	return &models.PerformanceMetrics{
		APILatencyMs:   apiLatency,
		ActiveSwitches: len(switches),
		TotalRequests:  hs.totalRequests,
		MemoryUsageMB:  float64(memStats.Alloc) / 1024 / 1024,
		DataAgeSeconds: avgDataAge,
		LastUpdate:     hs.lastFlushTime,
	}
}

// Background workers

// flushWorker handles background database writes
func (hs *HybridStore) flushWorker() {
	defer hs.wg.Done()

	for {
		select {
		case <-hs.ctx.Done():
			// Flush remaining items before shutdown with shorter timeout
			flushCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// Process remaining items in the queue quickly
			processed := 0
			for {
				select {
				case metrics := <-hs.flushQueue:
					if err := hs.writeToDatabase(flushCtx, metrics); err != nil {
						hs.logger.Warnf("Failed to flush metrics during shutdown: %v", err)
					} else {
						processed += len(metrics)
					}
				case <-flushCtx.Done():
					if processed > 0 {
						hs.logger.Infof("Flushed %d metrics during shutdown", processed)
					}
					return
				default:
					if processed > 0 {
						hs.logger.Infof("Flushed %d metrics during shutdown", processed)
					}
					return
				}
			}

		case metrics := <-hs.flushQueue:
			hs.writeToDatabase(hs.ctx, metrics)
		}
	}
}

// periodicFlushWorker triggers periodic cache cleanup and metrics reporting
func (hs *HybridStore) periodicFlushWorker() {
	defer hs.wg.Done()

	ticker := time.NewTicker(hs.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hs.ctx.Done():
			hs.logger.Debugf("Periodic flush worker shutting down")
			return
		case <-ticker.C:
			// Log performance metrics periodically
			metrics := hs.GetPerformanceMetrics()
			hs.logger.Infof("Telemetry performance: %s", metrics.String())
		}
	}
}

// cleanupWorker periodically cleans up stale cache entries
func (hs *HybridStore) cleanupWorker() {
	defer hs.wg.Done()

	ticker := time.NewTicker(hs.config.CacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-hs.ctx.Done():
			hs.logger.Debugf("Cleanup worker shutting down")
			return
		case <-ticker.C:
			cleaned := hs.cache.CleanupStale(hs.config.CacheTTL)
			if cleaned > 0 {
				hs.logger.Debugf("Cleaned up %d stale cache entries", cleaned)
			}
		}
	}
}

// writeToDatabase writes metrics to database with retry logic
func (hs *HybridStore) writeToDatabase(ctx context.Context, metrics []models.TelemetryData) error {
	if len(metrics) == 0 {
		return nil
	}

	var err error
	for attempt := 1; attempt <= hs.config.MaxRetries; attempt++ {
		// Check if context is cancelled before attempting database write
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Use bulk insert for better performance
		if repo, ok := hs.repository.(*PostgreSQLRepository); ok {
			err = repo.BulkStoreMetrics(ctx, metrics)
		} else {
			err = hs.repository.StoreMetrics(ctx, metrics)
		}

		if err == nil {
			hs.mu.Lock()
			hs.totalDBWrites += int64(len(metrics))
			hs.pendingFlushItems -= len(metrics)
			hs.lastFlushTime = time.Now()
			hs.mu.Unlock()

			hs.logger.Debugf("Successfully wrote %d metrics to database (attempt %d)", len(metrics), attempt)
			return nil
		}

		if attempt < hs.config.MaxRetries {
			backoff := time.Duration(attempt) * time.Second
			hs.logger.Warnf("Database write failed (attempt %d/%d), retrying in %v: %v",
				attempt, hs.config.MaxRetries, backoff, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	hs.mu.Lock()
	hs.totalDBWriteErrors++
	hs.mu.Unlock()

	return fmt.Errorf("failed to write metrics after %d attempts: %w", hs.config.MaxRetries, err)
}

// Performance tracking helpers
func (hs *HybridStore) incrementRequestCount() {
	hs.mu.Lock()
	hs.totalRequests++
	hs.mu.Unlock()
}

func (hs *HybridStore) incrementCacheHit() {
	hs.mu.Lock()
	hs.totalCacheHits++
	hs.mu.Unlock()
}
