package runner

import (
	"context"
	"time"

	"github.com/ufm/internal/app"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/monitoring/tracing"
	"github.com/ufm/internal/service"
	"github.com/ufm/internal/telemetry"
	"github.com/ufm/internal/telemetry/client"
	"github.com/ufm/internal/telemetry/models"
	"github.com/ufm/internal/telemetry/storage"
)

// TestInitializer provides a test-specific initializer that mocks external dependencies
type TestInitializer struct {
	// Mock configurations
	MockDatabaseURL     string
	MockGeneratorURL    string
	MockTelemetryConfig *config.TelemetryConfig
	MockServerConfig    *config.ServerConfig
	MockDatabaseConfig  *config.DatabaseConfig
}

// NewTestInitializer creates a new test initializer with default mock configurations
func NewTestInitializer() *TestInitializer {
	return &TestInitializer{
		MockDatabaseURL:  "postgres://test:test@localhost:5432/test_db?sslmode=disable",
		MockGeneratorURL: "http://localhost:9001",
		MockTelemetryConfig: &config.TelemetryConfig{
			Enabled: true,
			Ingestion: config.TelemetryIngestionConfig{
				Enabled:        true,
				GeneratorURL:   "http://localhost:9001",
				PollInterval:   "100ms",
				Timeout:        "5s",
				MaxRetries:     3,
				StartupDelay:   "1s",
				ReadinessCheck: false,
			},
			Queue: config.TelemetryQueueConfig{
				Enabled:   false,
				QueueSize: 1000,
				Workers:   5,
				Timeout:   "30s",
			},
		},
		MockServerConfig: &config.ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		MockDatabaseConfig: &config.DatabaseConfig{
			URL: "postgres://test:test@localhost:5432/test_db?sslmode=disable",
		},
	}
}

// InitServiceHome initializes the service home for testing
func (ti *TestInitializer) InitServiceHome(ctx context.Context, logger log.Logger) (context.Context, service.Home) {
	// Create a test-specific home directory
	testHome := service.NewServiceHome(ctx)

	// Create a test context with the home
	testCtx := service.NewContext(ctx, testHome, ti.createMockConfigService(logger), service.NewNodeInfo(), ti.createMockTracer(logger), ti.createMockLoggerFactory(logger))

	return testCtx, testHome
}

// InitServices initializes services with mocked external dependencies
func (ti *TestInitializer) InitServices(svcCtx service.Context) *app.AppServices {
	logger := svcCtx.LoggerFactory().(log.LoggerFactory).GetLogger("test-app")

	// Create mocked telemetry service
	telemetryService := ti.createMockTelemetryService(logger)

	// Create mocked generator client
	generatorClient := ti.createMockGeneratorClient(svcCtx, telemetryService, logger)

	return &app.AppServices{
		HttpServer:       nil,
		SystemHandler:    nil,
		TelemetryService: telemetryService,
		TelemetryHandler: nil,
		GeneratorClient:  generatorClient,
	}
}

// createMockConfigService creates a mock configuration service
func (ti *TestInitializer) createMockConfigService(logger log.Logger) config.Service {
	// Create a mock service home for testing
	mockHome := &MockServiceHome{}
	configService := config.NewService(context.Background(), logger, mockHome)
	return configService
}

// createMockTracer creates a mock tracer
func (ti *TestInitializer) createMockTracer(logger log.Logger) tracing.Tracer {
	// Return a simple mock tracer that does nothing
	return &MockTracer{
		logger: logger,
	}
}

// createMockLoggerFactory creates a mock logger factory
func (ti *TestInitializer) createMockLoggerFactory(logger log.Logger) interface{} {
	return &MockLoggerFactory{
		logger: logger,
	}
}

// createMockTelemetryService creates a mock telemetry service
func (ti *TestInitializer) createMockTelemetryService(logger log.Logger) telemetry.TelemetryService {
	// Create an in-memory cache for testing
	cache := storage.NewInMemoryCache()

	// Create a mock repository that doesn't actually connect to a database
	mockRepo := &MockRepository{}

	// Create hybrid store with mock components
	hybridConfig := storage.DefaultHybridStoreConfig()
	store := storage.NewHybridStore(cache, mockRepo, hybridConfig, logger)

	// Create telemetry service with mock store
	return telemetry.NewTelemetryService(store, logger)
}

// createMockGeneratorClient creates a mock generator client
func (ti *TestInitializer) createMockGeneratorClient(ctx service.Context, telemetryService telemetry.TelemetryService, logger log.Logger) client.GeneratorClientInterface {
	// Create a mock generator client that doesn't make real HTTP calls
	return &MockGeneratorClient{
		logger:           logger,
		telemetryService: telemetryService,
	}
}

// MockTracer is a mock implementation of the tracer interface
type MockTracer struct {
	logger log.Logger
}

func (m *MockTracer) StartSpanFromContext(ctx context.Context, operationName string) (context.Context, tracing.SpanCloseFunction) {
	m.logger.Debugf("Mock tracer: starting span %s", operationName)
	return ctx, func() {
		m.logger.Debugf("Mock tracer: finishing span %s", operationName)
	}
}

func (m *MockTracer) Close() error {
	m.logger.Debugf("Mock tracer: closing")
	return nil
}

// MockLoggerFactory is a mock implementation of the logger factory
type MockLoggerFactory struct {
	logger log.Logger
}

func (m *MockLoggerFactory) GetLogger(name string) log.Logger {
	return m.logger
}

func (m *MockLoggerFactory) GetRootLogger() log.Logger {
	return m.logger
}

func (m *MockLoggerFactory) GetRequestLogger() log.Logger {
	return m.logger
}

// MockRepository is a mock implementation of the repository interface
type MockRepository struct {
	data map[string]interface{}
}

func (m *MockRepository) CreateSwitch(ctx context.Context, sw models.Switch) error {
	// Mock create switch operation
	return nil
}

func (m *MockRepository) GetSwitch(ctx context.Context, switchID string) (*models.Switch, error) {
	// Mock get switch operation
	return &models.Switch{
		ID:       switchID,
		Name:     "Mock Switch",
		Location: "Mock Location",
		Created:  time.Now(),
	}, nil
}

func (m *MockRepository) ListSwitches(ctx context.Context) ([]models.Switch, error) {
	// Mock list switches operation
	return []models.Switch{}, nil
}

func (m *MockRepository) StoreMetrics(ctx context.Context, metrics []models.TelemetryData) error {
	// Mock store metrics operation
	return nil
}

func (m *MockRepository) GetLatestMetrics(ctx context.Context, switchID string) (*models.TelemetryData, error) {
	// Mock get latest metrics operation
	return &models.TelemetryData{
		SwitchID:       switchID,
		BandwidthMbps:  1000.0,
		LatencyMs:      2.5,
		PacketErrors:   0,
		UtilizationPct: 75.0,
		TemperatureC:   45.0,
		Timestamp:      time.Now(),
	}, nil
}

func (m *MockRepository) GetHistoricalMetrics(ctx context.Context, switchID string, from, to time.Time) ([]models.TelemetryData, error) {
	// Mock get historical metrics operation
	return []models.TelemetryData{}, nil
}

func (m *MockRepository) DeleteOldMetrics(ctx context.Context, olderThan time.Time) error {
	// Mock delete old metrics operation
	return nil
}

func (m *MockRepository) GetMetricsCount(ctx context.Context) (int64, error) {
	// Mock get metrics count operation
	return 0, nil
}

// MockServiceHome is a mock implementation of the service home
type MockServiceHome struct{}

func (m *MockServiceHome) HomeDir() string {
	return "/tmp/test-home"
}

func (m *MockServiceHome) LogDir() string {
	return "/tmp/test-logs"
}

func (m *MockServiceHome) DataDir() string {
	return "/tmp/test-data"
}

func (m *MockServiceHome) ConfigDir() string {
	return "/tmp/test-config"
}

func (m *MockServiceHome) SystemConfigFile() string {
	return "/tmp/test-system.yaml"
}

// MockGeneratorClient is a mock implementation of the generator client
type MockGeneratorClient struct {
	logger           log.Logger
	telemetryService telemetry.TelemetryService
	isRunning        bool
	stopChan         chan struct{}
}

func (m *MockGeneratorClient) Start(ctx context.Context) error {
	m.logger.Infof("Mock generator client: starting")
	m.isRunning = true
	m.stopChan = make(chan struct{})

	// Simulate some telemetry data generation
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.logger.Infof("Mock generator client: context cancelled")
				return
			case <-m.stopChan:
				m.logger.Infof("Mock generator client: stopped")
				return
			case <-ticker.C:
				if m.isRunning {
					// Generate mock telemetry data
					m.generateMockData()
				}
			}
		}
	}()

	return nil
}

func (m *MockGeneratorClient) Stop() error {
	m.logger.Infof("Mock generator client: stopping")
	m.isRunning = false
	if m.stopChan != nil {
		close(m.stopChan)
	}
	return nil
}

func (m *MockGeneratorClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running": m.isRunning,
		"type":    "mock",
	}
}

func (m *MockGeneratorClient) generateMockData() {
	// Generate mock telemetry data for testing
	m.logger.Debugf("Mock generator client: generating mock data")
	// This would create and ingest mock telemetry data
}

// NewTestApp creates a new test app with mocked dependencies
func NewTestApp(ctx context.Context, logger log.Logger) app.ExtendedApp {
	testInitializer := NewTestInitializer()
	return app.NewAppWithInitializer(ctx, logger, testInitializer)
}
