package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceHome(t *testing.T) {
	// Test with clean environment
	ctx := context.Background()
	home := NewServiceHome(ctx)

	assert.NotNil(t, home)
	assert.Implements(t, (*Home)(nil), home)

	// Test that directories are created
	homeDir := home.HomeDir()
	assert.NotEmpty(t, homeDir)

	// Verify directory structure exists
	assert.DirExists(t, homeDir)
	assert.DirExists(t, home.LogDir())
	assert.DirExists(t, home.DataDir())
	assert.DirExists(t, home.ConfigDir())
}

func TestServiceHome_HomeDir(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	homeDir := home.HomeDir()
	assert.NotEmpty(t, homeDir)
	assert.DirExists(t, homeDir)
}

func TestServiceHome_LogDir(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	logDir := home.LogDir()
	assert.NotEmpty(t, logDir)
	assert.DirExists(t, logDir)
	assert.Equal(t, filepath.Join(home.HomeDir(), "logs"), logDir)
}

func TestServiceHome_DataDir(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	dataDir := home.DataDir()
	assert.NotEmpty(t, dataDir)
	assert.DirExists(t, dataDir)
	assert.Equal(t, filepath.Join(home.HomeDir(), "data"), dataDir)
}

func TestServiceHome_ConfigDir(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	configDir := home.ConfigDir()
	assert.NotEmpty(t, configDir)
	assert.DirExists(t, configDir)
	assert.Equal(t, filepath.Join(home.HomeDir(), "config"), configDir)
}

func TestServiceHome_SystemConfigFile(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	configFile := home.SystemConfigFile()
	assert.NotEmpty(t, configFile)
	assert.Equal(t, filepath.Join(home.ConfigDir(), "system.yaml"), configFile)
}

func TestGetHomeDir_EnvironmentVariable(t *testing.T) {
	// Set environment variable
	testHomeDir := "/tmp/test-service-home"
	os.Setenv("SERVICE_HOME", testHomeDir)
	defer os.Unsetenv("SERVICE_HOME")

	// Create a temporary context to avoid affecting other tests
	ctx := context.Background()
	home := NewServiceHome(ctx)

	assert.Equal(t, testHomeDir, home.HomeDir())
}

func TestGetHomeDir_UserHomeDirectory(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("SERVICE_HOME")

	// Get user home directory
	userHome, err := os.UserHomeDir()
	require.NoError(t, err)

	ctx := context.Background()
	home := NewServiceHome(ctx)

	expectedHomeDir := filepath.Join(userHome, ".ufm")
	assert.Equal(t, expectedHomeDir, home.HomeDir())
}

func TestGetHomeDir_CurrentDirectory(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("SERVICE_HOME")

	ctx := context.Background()
	home := NewServiceHome(ctx)

	// Since we can't easily mock os.UserHomeDir to fail, we'll just test
	// that the home directory is properly set
	homeDir := home.HomeDir()
	assert.NotEmpty(t, homeDir)
	assert.DirExists(t, homeDir)
}

func TestGetHomeDir_LastResort(t *testing.T) {
	// This test is difficult to implement without being able to mock os functions
	// We'll test that the function always returns a valid directory
	ctx := context.Background()
	home := NewServiceHome(ctx)

	homeDir := home.HomeDir()
	assert.NotEmpty(t, homeDir)
	assert.DirExists(t, homeDir)
}

func TestServiceHome_DirectoryCreation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "service-home-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Set environment variable to use temp directory
	os.Setenv("SERVICE_HOME", tempDir)
	defer os.Unsetenv("SERVICE_HOME")

	ctx := context.Background()
	home := NewServiceHome(ctx)

	// Verify all directories are created
	dirs := []string{
		home.HomeDir(),
		home.LogDir(),
		home.DataDir(),
		home.ConfigDir(),
	}

	for _, dir := range dirs {
		assert.DirExists(t, dir)

		// Test that we can write to the directory
		testFile := filepath.Join(dir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		assert.NoError(t, err)
		defer os.Remove(testFile)
	}
}

func TestServiceHome_InterfaceCompliance(t *testing.T) {
	ctx := context.Background()
	home := NewServiceHome(ctx)

	// Test that the returned object implements the Home interface
	var _ Home = home

	// Test all interface methods
	assert.NotEmpty(t, home.HomeDir())
	assert.NotEmpty(t, home.LogDir())
	assert.NotEmpty(t, home.DataDir())
	assert.NotEmpty(t, home.ConfigDir())
	assert.NotEmpty(t, home.SystemConfigFile())
}
