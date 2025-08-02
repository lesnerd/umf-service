package main

import (
	"context"
	"os"

	"github.com/ufm/internal/app"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/service"
)

// @title UFM Service
// @description This is a Unified Fabric Manager service based on modern service architecture patterns.
// @version 1.0
// @schemes http
// @host localhost:8080
// @BasePath /api/v1 for system endpoints
func main() {
	ctx := context.Background()

	// Create a temporary logger for configuration loading
	tempLogger := log.DefaultLogger

	// Load configuration to get logging settings
	serviceHome := service.NewServiceHome(ctx)
	configService := config.NewService(ctx, tempLogger, serviceHome)
	cfg := configService.Get()

	logLevel := getLogLevel(cfg.Logging.Level)
	logger := log.NewLoggerWithConfig(
		logLevel,
		cfg.Logging.Format,
		os.Stdout,
	)

	application := app.NewApp(ctx, logger)
	application.Start()
	defer application.Stop()
	application.Wait()
}

func getLogLevel(configLevel string) string {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		return envLevel
	}
	return configLevel
}
