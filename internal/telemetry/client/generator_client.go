package client

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry"
	"github.com/ufm/internal/telemetry/models"
)

// GeneratorClientInterface defines the interface for generator client operations
type GeneratorClientInterface interface {
	Start(ctx context.Context) error
	Stop() error
	GetStats() map[string]interface{}
}

// GeneratorClient handles HTTP polling of the telemetry generator service
type GeneratorClient struct {
	generatorURL   string
	httpClient     *http.Client
	logger         log.Logger
	service        telemetry.TelemetryService
	pollInterval   time.Duration
	timeout        time.Duration
	maxRetries     int
	startupDelay   time.Duration
	readinessCheck bool

	// State management
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex

	// Deduplication tracking
	lastGenerationID  string
	lastDataTimestamp time.Time

	// Performance metrics
	totalPolls      int64
	successfulPolls int64
	duplicateSkips  int64
	errorCount      int64
	lastPollTime    time.Time
}

// GeneratorClientConfig holds the configuration for the generator client
type GeneratorClientConfig struct {
	GeneratorURL   string
	PollInterval   time.Duration
	Timeout        time.Duration
	MaxRetries     int
	StartupDelay   time.Duration
	ReadinessCheck bool
}

// NewGeneratorClient creates a new generator HTTP client
func NewGeneratorClient(
	config GeneratorClientConfig,
	service telemetry.TelemetryService,
	logger log.Logger,
) GeneratorClientInterface {
	if logger == nil {
		logger = log.DefaultLogger
	}

	// Create simple HTTP client optimized for 1-second polling
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,              // Connection pool
			MaxIdleConnsPerHost: 2,               // Keep few connections to generator
			IdleConnTimeout:     5 * time.Second, // Reasonable timeout
			DisableCompression:  true,            // CSV data doesn't benefit from compression
			DisableKeepAlives:   false,
		},
	}

	return &GeneratorClient{
		generatorURL:   config.GeneratorURL,
		httpClient:     httpClient,
		logger:         logger,
		service:        service,
		pollInterval:   config.PollInterval,
		timeout:        config.Timeout,
		maxRetries:     config.MaxRetries,
		startupDelay:   config.StartupDelay,
		readinessCheck: config.ReadinessCheck,
	}
}

// Start begins the polling process
func (gc *GeneratorClient) Start(ctx context.Context) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.running {
		return fmt.Errorf("generator client already running")
	}

	gc.ctx, gc.cancel = context.WithCancel(ctx)
	gc.running = true

	// Start the polling worker
	gc.wg.Add(1)
	go gc.pollingWorker()

	gc.logger.Infof("Generator client started, polling %s every %v",
		gc.generatorURL, gc.pollInterval)

	return nil
}

// Stop gracefully shuts down the generator client
func (gc *GeneratorClient) Stop() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if !gc.running {
		return fmt.Errorf("generator client not running")
	}

	gc.logger.Infof("Stopping generator client immediately...")
	gc.cancel()
	gc.running = false

	// Wait for worker to finish with a short timeout..
	done := make(chan struct{})
	go func() {
		gc.wg.Wait()
		close(done)
	}()

	// Wait up to 2 seconds for a gracefull shutdown
	select {
	case <-done:
		gc.logger.Infof("Generator client stopped gracefully")
	case <-time.After(2 * time.Second):
		gc.logger.Warnf("Generator client stop timed out, forcing shutdown")
	}

	// Close HTTP client to prevent hanging connections
	if gc.httpClient != nil {
		gc.httpClient.CloseIdleConnections()
	}

	return nil
}

// pollingWorker runs the background polling loop
func (gc *GeneratorClient) pollingWorker() {
	defer gc.wg.Done()

	// Wait for startup delay to ensure telemetry service is ready
	if gc.startupDelay > 0 {
		gc.logger.Infof("Waiting %v before starting to poll generator...", gc.startupDelay)
		select {
		case <-gc.ctx.Done():
			return
		case <-time.After(gc.startupDelay):
			// Continue after delay
		}
	}

	// Check generator readiness if enabled
	if gc.readinessCheck {
		if err := gc.checkGeneratorReadiness(); err != nil {
			gc.logger.Errorf("Generator readiness check failed: %v", err)
			// Continue anyway, but log the error
		} else {
			gc.logger.Infof("Generator readiness check passed")
		}
	}

	ticker := time.NewTicker(gc.pollInterval)
	defer ticker.Stop()

	// Do an initial poll
	gc.pollAndIngest()

	for {
		select {
		case <-gc.ctx.Done():
			return
		case <-ticker.C:
			gc.pollAndIngest()
		}
	}
}

// pollAndIngest polls the generator and ingests new data synchronously
func (gc *GeneratorClient) pollAndIngest() {
	gc.mu.Lock()
	gc.totalPolls++
	gc.lastPollTime = time.Now()
	gc.mu.Unlock()

	// Process data synchronously for immediate, reliable processing
	gc.fetchAndProcessData()
}

// fetchAndProcessData fetches and processes data synchronously
func (gc *GeneratorClient) fetchAndProcessData() {

	// Fetch data from generator (ufm)
	csvData, headers, err := gc.fetchCSVData()
	if err != nil {
		gc.mu.Lock()
		gc.errorCount++
		gc.mu.Unlock()
		gc.logger.Errorf("Failed to fetch CSV data: %v", err)
		return
	}

	generationID := headers.Get("X-Generation-ID")
	dataTimestamp := gc.parseTimestamp(headers.Get("X-Data-Timestamp"))

	// If no timestamp in headers, use first row timestamp from CSV
	if dataTimestamp.IsZero() && len(csvData) > 0 {
		dataTimestamp = gc.extractFirstTimestampFromCSV(csvData)
	}

	if gc.isDuplicateData(generationID, dataTimestamp) {
		gc.mu.Lock()
		gc.duplicateSkips++
		gc.mu.Unlock()
		gc.logger.Debugf("Skipping duplicate data (generation_id: %s, timestamp: %v)", generationID, dataTimestamp.Format(time.RFC3339))
		return
	}

	// Parse and ingest the CSV data
	telemetryData, err := gc.parseCSVData(csvData)
	if err != nil {
		gc.mu.Lock()
		gc.errorCount++
		gc.mu.Unlock()
		gc.logger.Errorf("Failed to parse CSV data: %v", err)
		return
	}

	// Register switches from telemetry data
	gc.registerSwitchesFromData(telemetryData)

	// Ingest the data
	if err := gc.service.IngestBatch(telemetryData); err != nil {
		gc.mu.Lock()
		gc.errorCount++
		gc.mu.Unlock()
		gc.logger.Errorf("Failed to ingest telemetry data: %v", err)
		return
	}

	// Update deduplication tracking
	gc.mu.Lock()
	gc.lastGenerationID = generationID
	gc.lastDataTimestamp = dataTimestamp
	gc.successfulPolls++
	gc.mu.Unlock()

	gc.logger.Infof("Successfully ingested %d telemetry records, generation_id=%s", len(telemetryData), generationID)
}

// fetchCSVData fetches CSV data from the generator with simple error handling
func (gc *GeneratorClient) fetchCSVData() (string, http.Header, error) {
	url := gc.generatorURL + "/counters"

	// Check if context is cancelled before making request
	select {
	case <-gc.ctx.Done():
		return "", nil, gc.ctx.Err()
	default:
	}

	ctx, cancel := context.WithTimeout(gc.ctx, gc.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for better HTTP performance
	req.Header.Set("Accept", "text/csv")
	req.Header.Set("User-Agent", "UFM-Telemetry-Client/1.0")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		// Simple error logging - don't retry, just continue to next poll
		gc.logger.Warnf("HTTP request failed: %v", err)
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		gc.logger.Warnf("HTTP status error: %v", err)
		return "", nil, err
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		gc.logger.Warnf("Failed to read response body: %v", err)
		return "", nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), resp.Header, nil
}

// checkGeneratorReadiness verifies that the generator is accessible and healthy
func (gc *GeneratorClient) checkGeneratorReadiness() error {
	url := gc.generatorURL + "/health"

	ctx, cancel := context.WithTimeout(gc.ctx, gc.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create readiness check request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("generator health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("generator health check returned status %d", resp.StatusCode)
	}

	return nil
}

// checkGeneratorConnectivity performs a quick connectivity check
func (gc *GeneratorClient) checkGeneratorConnectivity() error {
	url := gc.generatorURL + "/counters"

	ctx, cancel := context.WithTimeout(gc.ctx, 5*time.Second) // Quick 5s check
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create connectivity check request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("generator connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("generator connectivity check returned status %d", resp.StatusCode)
	}

	return nil
}

// parseCSVData parses CSV data into telemetry data structures
func (gc *GeneratorClient) parseCSVData(csvData string) ([]models.TelemetryData, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 { // Header + at least one data row
		return nil, fmt.Errorf("CSV contains no data rows")
	}

	// Skip header row
	var telemetryData []models.TelemetryData
	for i, record := range records[1:] {
		if len(record) < 7 {
			gc.logger.Warnf("Skipping CSV row %d: insufficient columns (%d)", i+2, len(record))
			continue
		}

		data, err := gc.parseCSVRecord(record)
		if err != nil {
			gc.logger.Warnf("Skipping CSV row %d: %v", i+2, err)
			continue
		}

		telemetryData = append(telemetryData, data)
	}

	return telemetryData, nil
}

// registerSwitchesFromData registers switches found in telemetry data
func (gc *GeneratorClient) registerSwitchesFromData(telemetryData []models.TelemetryData) {
	// Track unique switches to avoid duplicate registrations
	seenSwitches := make(map[string]bool)

	for _, data := range telemetryData {
		if data.SwitchID == "" {
			continue
		}

		// Skip if we've already processed this switch in this batch
		if seenSwitches[data.SwitchID] {
			continue
		}
		seenSwitches[data.SwitchID] = true

		// Create switch record with "data center" as location
		switchRecord := models.Switch{
			ID:       data.SwitchID,
			Name:     data.SwitchID,
			Location: "data center",
			Created:  time.Now(),
		}

		// Register the switch
		if err := gc.service.RegisterSwitch(switchRecord); err != nil {
			gc.logger.Warnf("Failed to register switch %s: %v", data.SwitchID, err)
		} else {
			gc.logger.Debugf("Registered switch: %s", data.SwitchID)
		}
	}
}

// parseCSVRecord parses a single CSV record into TelemetryData
func (gc *GeneratorClient) parseCSVRecord(record []string) (models.TelemetryData, error) {
	// CSV format: switch_id,timestamp,bandwidth_mbps,latency_ms,packet_errors,utilization_pct,temperature_c
	timestamp, err := time.Parse(time.RFC3339Nano, record[1])
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	bandwidth, err := strconv.ParseFloat(record[2], 64)
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid bandwidth: %w", err)
	}

	latency, err := strconv.ParseFloat(record[3], 64)
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid latency: %w", err)
	}

	packetErrors, err := strconv.ParseInt(record[4], 10, 64)
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid packet errors: %w", err)
	}

	utilization, err := strconv.ParseFloat(record[5], 64)
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid utilization: %w", err)
	}

	temperature, err := strconv.ParseFloat(record[6], 64)
	if err != nil {
		return models.TelemetryData{}, fmt.Errorf("invalid temperature: %w", err)
	}

	return models.TelemetryData{
		SwitchID:       record[0],
		Timestamp:      timestamp,
		BandwidthMbps:  bandwidth,
		LatencyMs:      latency,
		PacketErrors:   packetErrors,
		UtilizationPct: utilization,
		TemperatureC:   temperature,
	}, nil
}

// isDuplicateData checks if the data has already been processed
func (gc *GeneratorClient) isDuplicateData(generationID string, dataTimestamp time.Time) bool {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	// Check generation ID first (most reliable)
	if generationID != "" && generationID == gc.lastGenerationID {
		return true
	}

	// Fallback to timestamp comparison
	if !dataTimestamp.IsZero() && !dataTimestamp.After(gc.lastDataTimestamp) {
		return true
	}

	return false
}

// parseTimestamp safely parses a timestamp string
func (gc *GeneratorClient) parseTimestamp(timestampStr string) time.Time {
	if timestampStr == "" {
		return time.Time{}
	}

	timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		gc.logger.Debugf("Failed to parse timestamp '%s': %v", timestampStr, err)
		return time.Time{}
	}

	return timestamp
}

// extractFirstTimestampFromCSV extracts timestamp from the first data row of CSV
func (gc *GeneratorClient) extractFirstTimestampFromCSV(csvData string) time.Time {
	lines := strings.Split(csvData, "\n")
	if len(lines) < 2 { // Need header + at least one data row
		return time.Time{}
	}

	// Skip header
	fields := strings.Split(lines[1], ",")
	if len(fields) < 2 {
		return time.Time{}
	}

	timestamp, err := time.Parse(time.RFC3339Nano, fields[1])
	if err != nil {
		gc.logger.Debugf("Failed to parse CSV timestamp '%s': %v", fields[1], err)
		return time.Time{}
	}

	return timestamp
}

// GetStats returns performance statistics
func (gc *GeneratorClient) GetStats() map[string]interface{} {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	successRate := float64(0)
	if gc.totalPolls > 0 {
		successRate = float64(gc.successfulPolls) / float64(gc.totalPolls) * 100
	}

	duplicateRate := float64(0)
	if gc.totalPolls > 0 {
		duplicateRate = float64(gc.duplicateSkips) / float64(gc.totalPolls) * 100
	}

	return map[string]interface{}{
		"total_polls":         gc.totalPolls,
		"successful_polls":    gc.successfulPolls,
		"duplicate_skips":     gc.duplicateSkips,
		"error_count":         gc.errorCount,
		"success_rate":        fmt.Sprintf("%.2f%%", successRate),
		"duplicate_rate":      fmt.Sprintf("%.2f%%", duplicateRate),
		"last_poll_time":      gc.lastPollTime.Format(time.RFC3339),
		"last_generation_id":  gc.lastGenerationID,
		"last_data_timestamp": gc.lastDataTimestamp.Format(time.RFC3339Nano),
		"poll_interval":       gc.pollInterval.String(),
	}
}
