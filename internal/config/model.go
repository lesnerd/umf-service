package config

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Logging   LoggingConfig   `yaml:"logging"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Tracing   TracingConfig   `yaml:"tracing"`
	Security  SecurityConfig  `yaml:"security"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
}

type ServerConfig struct {
	Port        string `yaml:"port" env:"SERVER_PORT"`
	Host        string `yaml:"host" env:"SERVER_HOST"`
	Timeout     int    `yaml:"timeout"`
	Multitenant bool   `yaml:"multitenant"`
}

type DatabaseConfig struct {
	Type                      string `yaml:"type" env:"DATABASE_TYPE"`
	URL                       string `yaml:"url" env:"DATABASE_URL"`
	Username                  string `yaml:"username" env:"DATABASE_USERNAME"`
	Password                  string `yaml:"password" env:"DATABASE_PASSWORD"`
	QueryTimeoutSecs          int    `yaml:"queryTimeoutSecs"`
	MaxIdleConnections        int    `yaml:"maxIdleConnections"`
	MaxOpenConnections        int    `yaml:"maxOpenConnections"`
	ConnectionMaxLifetimeSecs int    `yaml:"connectionMaxLifetimeSecs"`
}

type LoggingConfig struct {
	Level    string `yaml:"level" env:"LOG_LEVEL"`
	Format   string `yaml:"format" env:"LOG_FORMAT"`
	Console  bool   `yaml:"console"`
	FilePath string `yaml:"filePath"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" env:"METRICS_ENABLED"`
	Port    string `yaml:"port" env:"METRICS_PORT"`
	Path    string `yaml:"path"`
}

type TracingConfig struct {
	Enabled     bool   `yaml:"enabled" env:"TRACING_ENABLED"`
	ServiceName string `yaml:"serviceName"`
	Endpoint    string `yaml:"endpoint" env:"TRACING_ENDPOINT"`
}

type SecurityConfig struct {
	JWTSecret      string          `yaml:"jwtSecret" env:"JWT_SECRET"`
	AllowedOrigins []string        `yaml:"allowedOrigins"`
	RateLimiting   RateLimitConfig `yaml:"rateLimiting"`
}

type RateLimitConfig struct {
	Enabled bool `yaml:"enabled"`
	RPS     int  `yaml:"rps"`
	Burst   int  `yaml:"burst"`
}

type TelemetryConfig struct {
	Enabled   bool                     `yaml:"enabled" env:"TELEMETRY_ENABLED"`
	Storage   TelemetryStorageConfig   `yaml:"storage"`
	Queue     TelemetryQueueConfig     `yaml:"queue"`
	Simulator TelemetrySimulatorConfig `yaml:"simulator"`
	Ingestion TelemetryIngestionConfig `yaml:"ingestion"`
}

type TelemetryStorageConfig struct {
	CacheTTL      string `yaml:"cache_ttl"`
	BatchSize     int    `yaml:"batch_size"`
	FlushInterval string `yaml:"flush_interval"`
	MaxRetries    int    `yaml:"max_retries"`
}

type TelemetryQueueConfig struct {
	Enabled   bool   `yaml:"enabled"`
	QueueSize int    `yaml:"queue_size"`
	Workers   int    `yaml:"workers"`
	Timeout   string `yaml:"timeout"`
}

type TelemetrySimulatorConfig struct {
	SwitchCount    int    `yaml:"switch_count"`
	UpdateInterval string `yaml:"update_interval"`
	EnableSpikes   bool   `yaml:"enable_spikes"`
	EnableErrors   bool   `yaml:"enable_errors"`
}

type TelemetryIngestionConfig struct {
	Enabled        bool   `yaml:"enabled" env:"TELEMETRY_INGESTION_ENABLED"`
	GeneratorURL   string `yaml:"generator_url" env:"TELEMETRY_GENERATOR_URL"`
	PollInterval   string `yaml:"poll_interval" env:"TELEMETRY_POLL_INTERVAL"`
	Timeout        string `yaml:"timeout" env:"TELEMETRY_TIMEOUT"`
	MaxRetries     int    `yaml:"max_retries" env:"TELEMETRY_MAX_RETRIES"`
	StartupDelay   string `yaml:"startup_delay" env:"TELEMETRY_STARTUP_DELAY"`
	ReadinessCheck bool   `yaml:"readiness_check" env:"TELEMETRY_READINESS_CHECK"`
}
