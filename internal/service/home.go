package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/ufm/internal/shared"
)

// Re-export the shared Home interface
type Home = shared.Home

type serviceHome struct {
	homeDir   string
	logDir    string
	dataDir   string
	configDir string
}

func NewServiceHome(ctx context.Context) Home {
	homeDir := getHomeDir()

	// Create the home directory structure
	dirs := []string{
		homeDir,
		filepath.Join(homeDir, "logs"),
		filepath.Join(homeDir, "data"),
		filepath.Join(homeDir, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Log error but continue - the sysconfig will handle directory creation
			// when it tries to set up the file watcher
		}
	}

	return &serviceHome{
		homeDir:   homeDir,
		logDir:    filepath.Join(homeDir, "logs"),
		dataDir:   filepath.Join(homeDir, "data"),
		configDir: filepath.Join(homeDir, "config"),
	}
}

func (h *serviceHome) HomeDir() string {
	return h.homeDir
}

func (h *serviceHome) LogDir() string {
	return h.logDir
}

func (h *serviceHome) DataDir() string {
	return h.dataDir
}

func (h *serviceHome) ConfigDir() string {
	return h.configDir
}

func (h *serviceHome) SystemConfigFile() string {
	return filepath.Join(h.configDir, "system.yaml")
}

func getHomeDir() string {
	// Try environment variable first
	if homeDir := os.Getenv("SERVICE_HOME"); homeDir != "" {
		return homeDir
	}

	// Try user home directory
	if userHome, err := os.UserHomeDir(); err == nil {
		return filepath.Join(userHome, ".ufm")
	}

	// Fallback to current directory
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, ".service")
	}

	// Last resort
	return "/tmp/ufm"
}
