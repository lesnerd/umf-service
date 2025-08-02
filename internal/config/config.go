package config

import (
	"context"
	"fmt"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/shared"
	"github.com/ufm/internal/sysconfig"
)

type Service interface {
	Get() Config
	// AddUpdateListener - The listeners are triggered when the config *file* is updated
	AddUpdateListener(listener UpdateListener)
	IsMultiTenant() bool
	GetHomeDir() string
}

type configService struct {
	config          *Config
	serviceHome     shared.Home
	updateListeners []UpdateListener
	sysConfig       *sysconfig.Config
}

func (s *configService) GetHomeDir() string {
	return s.serviceHome.HomeDir()
}

type UpdateListener struct {
	Name     string
	OnUpdate func(context.Context, Config) error
}

func NewService(ctx context.Context, logger log.Logger, serviceHome shared.Home) Service {
	s := configService{
		config:          &Config{},
		serviceHome:     serviceHome,
		updateListeners: []UpdateListener{},
	}
	if err := s.init(ctx, logger); err != nil {
		logger.Fatalf("Failed to initialize configuration: %+v", err)
	}
	return &s
}

func (s *configService) Get() Config {
	return *s.config
}

func (s *configService) AddUpdateListener(listener UpdateListener) {
	s.updateListeners = append(s.updateListeners, listener)
}

func (s *configService) init(ctx context.Context, logger log.Logger) error {
	sysConfig, err := sysconfig.Load(s.serviceHome.SystemConfigFile(),
		sysconfig.WithDefaultValues(getConfigDefaults()),
		sysconfig.WithLogger(logger),
		sysconfig.WithUpdateListeners([]sysconfig.UpdateListenersFunc{
			s.sysConfigUpdateListener(ctx, logger),
		}),
		sysconfig.WithFileWatcher(sysconfig.FileWatchConfig{
			Context:       ctx,
			UpdatableKeys: []string{KeyApplicationLogLevel, ".ufm.*"},
		}))
	if err != nil {
		return err
	}
	s.sysConfig = sysConfig
	return s.unmarshalConfig()
}

func (s *configService) validateConfig() error {
	if s.config.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if s.config.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	return nil
}

func (s *configService) triggerUpdateListeners(ctx context.Context, logger log.Logger) {
	if len(s.updateListeners) > 0 {
		logger.Infof("Configuration update detected")
	}
	for _, listener := range s.updateListeners {
		logger.Debugf("Running config update listener '%s'", listener.Name)
		if err := listener.OnUpdate(ctx, *s.config); err != nil {
			logger.Errorf("Failed running config update listener '%s': %v", listener.Name, err)
		}
		logger.Debugf("Successfully ran config update listener '%s'", listener.Name)
	}
}

func (s *configService) IsMultiTenant() bool {
	return s.Get().Server.Multitenant
}

func (s *configService) sysConfigUpdateListener(ctx context.Context, logger log.Logger) func(context.Context, map[string]interface{}) error {
	return func(ctx context.Context, data map[string]interface{}) error {
		logger.Infof("System configuration updated")

		// Update the configuration from the new data
		if err := s.unmarshalConfig(); err != nil {
			return fmt.Errorf("failed to unmarshal config after update: %w", err)
		}

		// Trigger existing update listeners
		s.triggerUpdateListeners(ctx, logger)

		return nil
	}
}

func (s *configService) unmarshalConfig() error {
	if s.sysConfig == nil {
		return fmt.Errorf("system configuration not initialized")
	}

	config := Config{
		Server: ServerConfig{
			Port:        s.getStringOrDefault("server.port", "8080"),
			Host:        s.getStringOrDefault("server.host", "localhost"),
			Timeout:     s.getIntOrDefault("server.timeout", 30),
			Multitenant: s.getBoolOrDefault("server.multitenant", false),
		},
		Database: DatabaseConfig{
			Type:               s.getStringOrDefault("database.type", "postgres"),
			URL:                s.getStringOrDefault("database.url", "postgres://umf_user:umf_password@localhost:5432/umf_db?sslmode=disable"),
			MaxOpenConnections: s.getIntOrDefault("database.maxOpenConnections", 10),
			MaxIdleConnections: s.getIntOrDefault("database.maxIdleConnections", 5),
			QueryTimeoutSecs:   s.getIntOrDefault("database.queryTimeoutSecs", 30),
		},
		Logging: LoggingConfig{
			Level:   s.getStringOrDefault("logging.level", "info"),
			Format:  s.getStringOrDefault("logging.format", "text"),
			Console: s.getBoolOrDefault("logging.console", true),
		},
		Metrics: MetricsConfig{
			Enabled: s.getBoolOrDefault("metrics.enabled", false),
			Port:    s.getStringOrDefault("metrics.port", "9090"),
		},
		Tracing: TracingConfig{
			Enabled:     s.getBoolOrDefault("tracing.enabled", false),
			ServiceName: s.getStringOrDefault("tracing.serviceName", "ufm"),
		},
		Telemetry: TelemetryConfig{
			Enabled: s.getBoolOrDefault("telemetry.enabled", true),
			Storage: TelemetryStorageConfig{
				CacheTTL:      s.getStringOrDefault("telemetry.storage.cache_ttl", "5m"),
				BatchSize:     s.getIntOrDefault("telemetry.storage.batch_size", 100),
				FlushInterval: s.getStringOrDefault("telemetry.storage.flush_interval", "30s"),
				MaxRetries:    s.getIntOrDefault("telemetry.storage.max_retries", 3),
			},
			Queue: TelemetryQueueConfig{
				Enabled:   s.getBoolOrDefault("telemetry.queue.enabled", false),
				QueueSize: s.getIntOrDefault("telemetry.queue.queue_size", 1000),
				Workers:   s.getIntOrDefault("telemetry.queue.workers", 10),
				Timeout:   s.getStringOrDefault("telemetry.queue.timeout", "5s"),
			},
			Simulator: TelemetrySimulatorConfig{
				SwitchCount:    s.getIntOrDefault("telemetry.simulator.switch_count", 10),
				UpdateInterval: s.getStringOrDefault("telemetry.simulator.update_interval", "10s"),
				EnableSpikes:   s.getBoolOrDefault("telemetry.simulator.enable_spikes", true),
				EnableErrors:   s.getBoolOrDefault("telemetry.simulator.enable_errors", true),
			},
			Ingestion: TelemetryIngestionConfig{
				Enabled:      s.getBoolOrDefault("telemetry.ingestion.enabled", true),
				GeneratorURL: s.getStringOrDefault("telemetry.ingestion.generator_url", "http://localhost:9001"),
				PollInterval: s.getStringOrDefault("telemetry.ingestion.poll_interval", "1s"),
				Timeout:      s.getStringOrDefault("telemetry.ingestion.timeout", "10s"),
				MaxRetries:   s.getIntOrDefault("telemetry.ingestion.max_retries", 3),
			},
		},
	}

	s.config = &config
	return nil
}

func (s *configService) getStringOrDefault(key, defaultValue string) string {
	if val := s.sysConfig.GetString(key); val != "" {
		return val
	}
	return defaultValue
}

func (s *configService) getIntOrDefault(key string, defaultValue int) int {
	if val := s.sysConfig.GetInt(key); val != 0 {
		return val
	}
	return defaultValue
}

func (s *configService) getBoolOrDefault(key string, defaultValue bool) bool {
	if s.sysConfig.Get(key) != nil {
		return s.sysConfig.GetBool(key)
	}
	return defaultValue
}

const (
	KeyApplicationLogLevel = "application.log.level"
)

func getConfigDefaults() map[string]interface{} {
	// IMPORTANT: These are fallback defaults only - YAML values should take precedence
	// Keep minimal defaults that don't conflict with typical YAML configurations
	return map[string]interface{}{
		// Core server defaults (essential for startup)
		"server.port":        "8080",
		"server.host":        "localhost",
		"server.timeout":     30,
		"server.multitenant": false,

		// Database defaults (essential for telemetry)
		"database.type":               "postgres",
		"database.url":                "postgres://umf_user:umf_password@localhost:5432/umf_db?sslmode=disable",
		"database.maxOpenConnections": 10,
		"database.maxIdleConnections": 5,
		"database.queryTimeoutSecs":   30,

		"logging.level":   "info",
		"logging.format":  "pretty",
		"logging.console": true,

		// Metrics defaults
		"metrics.enabled": false,
		"metrics.port":    "9090",

		// Tracing defaults
		"tracing.enabled":     false,
		"tracing.serviceName": "ufm",

		// Essential telemetry defaults only
		"telemetry.enabled":                   true,
		"telemetry.ingestion.enabled":         true,
		"telemetry.ingestion.generator_url":   "http://localhost:9001",
		"telemetry.ingestion.poll_interval":   "1s",
		"telemetry.ingestion.timeout":         "5s",
		"telemetry.ingestion.max_retries":     3,
		"telemetry.ingestion.startup_delay":   "2s",
		"telemetry.ingestion.readiness_check": true,

		// Storage defaults (minimal settings)
		"telemetry.storage.cache_ttl":      "5m",
		"telemetry.storage.batch_size":     100,
		"telemetry.storage.flush_interval": "30s",
		"telemetry.storage.max_retries":    3,
	}
}
