package sysconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ufm/internal/log"
	"gopkg.in/yaml.v3"
)

// UpdateListenersFunc is a function that gets called when config is updated
type UpdateListenersFunc func(ctx context.Context, config map[string]interface{}) error

// FileWatchConfig configures file watching behavior
type FileWatchConfig struct {
	Context       context.Context
	UpdatableKeys []string
}

// Config represents the system configuration
type Config struct {
	logger          log.Logger
	configFile      string
	data            map[string]interface{}
	updateListeners []UpdateListenersFunc
	watcher         *fsnotify.Watcher
	watchConfig     *FileWatchConfig
}

// Option configures the system config
type Option func(*Config) error

// WithDefaultValues sets default configuration values
func WithDefaultValues(defaults map[string]interface{}) Option {
	return func(c *Config) error {
		for key, value := range defaults {
			c.data[key] = value
		}
		return nil
	}
}

// WithLogger sets the logger for the configuration
func WithLogger(logger log.Logger) Option {
	return func(c *Config) error {
		c.logger = logger
		return nil
	}
}

// WithUpdateListeners sets update listeners
func WithUpdateListeners(listeners []UpdateListenersFunc) Option {
	return func(c *Config) error {
		c.updateListeners = listeners
		return nil
	}
}

// WithFileWatcher enables file watching
func WithFileWatcher(watchConfig FileWatchConfig) Option {
	return func(c *Config) error {
		c.watchConfig = &watchConfig
		return nil
	}
}

// Load loads configuration from file with options
func Load(configFile string, options ...Option) (*Config, error) {
	config := &Config{
		configFile: configFile,
		data:       make(map[string]interface{}),
	}

	// Apply options
	for _, option := range options {
		if err := option(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Load configuration from file if it exists
	if err := config.loadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	// Start file watcher if configured
	if config.watchConfig != nil {
		if err := config.startWatcher(); err != nil {
			return nil, fmt.Errorf("failed to start file watcher: %w", err)
		}
	}

	return config, nil
}

// Get returns configuration value by key
func (c *Config) Get(key string) interface{} {
	return c.data[key]
}

// GetString returns configuration value as string
func (c *Config) GetString(key string) string {
	if val, ok := c.data[key].(string); ok {
		return val
	}
	return ""
}

// GetBool returns configuration value as bool
func (c *Config) GetBool(key string) bool {
	if val, ok := c.data[key].(bool); ok {
		return val
	}
	return false
}

// GetInt returns configuration value as int
func (c *Config) GetInt(key string) int {
	if val, ok := c.data[key].(float64); ok {
		return int(val)
	}
	return 0
}

// loadFromFile loads configuration from YAML file
func (c *Config) loadFromFile() error {
	if _, err := os.Stat(c.configFile); os.IsNotExist(err) {
		if c.logger != nil {
			c.logger.Debugf("Config file %s does not exist, using defaults", c.configFile)
		}
		return nil
	}

	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var fileData map[string]interface{}
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// Flatten nested YAML structure into dotted keys and merge with existing data
	flattenedData := flattenMap(fileData, "")
	for key, value := range flattenedData {
		c.data[key] = value
	}

	if c.logger != nil {
		c.logger.Debugf("Loaded configuration from %s", c.configFile)
	}

	return nil
}

// flattenMap converts nested map structures to flat dotted keys
// Example: {"logging": {"level": "debug"}} -> {"logging.level": "debug"}
func flattenMap(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		var newKey string
		if prefix == "" {
			newKey = key
		} else {
			newKey = prefix + "." + key
		}
		
		// If value is a nested map, flatten it recursively
		if nestedMap, ok := value.(map[string]interface{}); ok {
			for k, v := range flattenMap(nestedMap, newKey) {
				result[k] = v
			}
		} else {
			// For non-map values, store directly
			result[newKey] = value
		}
	}
	
	return result
}

// startWatcher starts file watching
func (c *Config) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	c.watcher = watcher

	// Watch the config file directory
	configDir := filepath.Dir(c.configFile)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	go c.handleWatchEvents()

	if c.logger != nil {
		c.logger.Debugf("Started file watcher for %s", c.configFile)
	}

	return nil
}

// handleWatchEvents handles file system events
func (c *Config) handleWatchEvents() {
	defer c.watcher.Close()

	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}

			if event.Name == c.configFile && (event.Op&fsnotify.Write == fsnotify.Write) {
				if c.logger != nil {
					c.logger.Debugf("Config file changed: %s", event.Name)
				}

				// Debounce multiple events
				time.Sleep(100 * time.Millisecond)

				if err := c.reloadConfig(); err != nil {
					if c.logger != nil {
						c.logger.Errorf("Failed to reload config: %v", err)
					}
				}
			}

		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
			if c.logger != nil {
				c.logger.Errorf("File watcher error: %v", err)
			}

		case <-c.watchConfig.Context.Done():
			return
		}
	}
}

// reloadConfig reloads configuration and triggers listeners
func (c *Config) reloadConfig() error {
	if err := c.loadFromFile(); err != nil {
		return err
	}

	// Trigger update listeners
	for _, listener := range c.updateListeners {
		if err := listener(c.watchConfig.Context, c.data); err != nil {
			if c.logger != nil {
				c.logger.Errorf("Update listener failed: %v", err)
			}
		}
	}

	return nil
}

// Close closes the configuration and stops file watching
func (c *Config) Close() error {
	if c.watcher != nil {
		return c.watcher.Close()
	}
	return nil
}
