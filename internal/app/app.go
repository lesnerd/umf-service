package app

import (
	"context"
	"database/sql"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/http"
	"github.com/ufm/internal/http/handler"
	"github.com/ufm/internal/http/middleware"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/monitoring/tracing"
	"github.com/ufm/internal/service"
	"github.com/ufm/internal/telemetry"
	"github.com/ufm/internal/telemetry/client"
	"github.com/ufm/internal/telemetry/queue"
	"github.com/ufm/internal/telemetry/storage"
)

const (
	operationName           = "bootstrap"
	ctxKeyStartTime         = "app.startTime"
	gracefulShutdownTimeout = 10 * time.Second // Reduced to 10s for faster shutdown
)

type App interface {
	Start()
	Wait()
	Stop()
}

type ExtendedApp interface {
	App
	GetServiceContext() service.Context
	GetServices() *AppServices
}

type Initializer interface {
	InitServiceHome(context.Context, log.Logger) (context.Context, service.Home)
	InitServices(service.Context) *AppServices
}

type app struct {
	ctx         service.Context
	logger      log.Logger
	services    *AppServices
	cancelFuncs []context.CancelFunc
}

type AppServices struct {
	HttpServer       http.Server
	SystemHandler    handler.SystemHandler
	TelemetryService telemetry.TelemetryService
	TelemetryHandler handler.TelemetryHandler
	GeneratorClient  *client.GeneratorClient
}

type initializer struct{}

func (a *app) GetServiceContext() service.Context {
	return a.ctx
}

func (a *app) GetServices() *AppServices {
	return a.services
}

func (i *initializer) InitServiceHome(ctx context.Context, logger log.Logger) (context.Context, service.Home) {
	serviceHome := service.NewServiceHome(ctx)
	return ctx, serviceHome
}

func (i *initializer) InitServices(svcCtx service.Context) *AppServices {
	return newServices(svcCtx)
}

func NewApp(ctx context.Context, logger log.Logger) App {
	return NewAppWithInitializer(ctx, logger, &initializer{})
}

func NewAppWithInitializer(ctx context.Context, logger log.Logger, serviceInitializer Initializer) ExtendedApp {
	ctx = context.WithValue(ctx, ctxKeyStartTime, time.Now())
	tracer := tracing.NewTracer(service.Type, logger)
	ctx, spanClose := tracer.StartSpanFromContext(ctx, operationName)
	defer spanClose()

	ctx, serviceHome := serviceInitializer.InitServiceHome(ctx, logger)

	logVersion(logger, serviceHome)
	configService := config.NewService(ctx, logger, serviceHome)
	loggingConfig := log.LoggingConfig{
		Level:    configService.Get().Logging.Level,
		Format:   configService.Get().Logging.Format,
		Console:  configService.Get().Logging.Console,
		FilePath: configService.Get().Logging.FilePath,
	}
	loggerFactory := log.NewLoggerFactory(ctx, logger, loggingConfig)
	logger = loggerFactory.GetLogger(operationName)
	logSystemConfig(logger, configService)

	nodeInfo := service.NewNodeInfo()
	svcCtx := service.NewContext(ctx, serviceHome, configService, nodeInfo, tracer, loggerFactory)
	logNodeInfo(logger, nodeInfo, configService)

	services := serviceInitializer.InitServices(svcCtx)
	return &app{
		ctx:         svcCtx,
		logger:      svcCtx.LoggerFactory().(log.LoggerFactory).GetLogger("app"),
		services:    services,
		cancelFuncs: []context.CancelFunc{},
	}
}

func (a *app) Start() {
	if err := a.services.TelemetryService.Start(a.ctx); err != nil {
		a.logger.Errorf("Failed to start telemetry service: %v", err)
	} else {
		a.logger.Infof("Telemetry service started successfully")
	}

	if err := a.services.GeneratorClient.Start(a.ctx); err != nil {
		a.logger.Errorf("Failed to start generator client: %v", err)
	} else {
		a.logger.Infof("Generator client started successfully")
	}

	httpAddr, err := a.services.HttpServer.Listen(a.ctx)
	if err != nil {
		a.logger.Fatalf("Failed to initialize HTTP listener: %v", err)
	}
	a.logger.Infof("HTTP server listening on %v", httpAddr)

	a.logInitCompleted()
	go a.startHttpServer()
}

func (a *app) startHttpServer() {
	err := a.services.HttpServer.Serve(&http.ServeConfig{
		InitRoutes: func(engine *gin.Engine) []io.Closer {
			http.RegisterHandlers(engine, a.services.SystemHandler, a.services.TelemetryHandler)
			return []io.Closer{}
		},
	})
	if err != nil {
		a.logger.Errorf("Error while serving HTTP: %v", err)
	}
}

func (a *app) Wait() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, os.Kill)
	sig := <-stopChan
	a.logger.Infof("Received OS signal: %v", sig)
}

func (a *app) Stop() {
	defer a.ctx.Done()
	ctx, cancel := context.WithTimeout(a.ctx, gracefulShutdownTimeout)
	a.logger.Infof("Starting graceful shutdown (timeout: %v)...", gracefulShutdownTimeout)
	defer cancel()

	// Step 1: Stop fetching from localhost:9001 immediately so no new data is fetched
	a.logger.Infof("Step 1: Stopping data fetching from generator...")
	if err := a.services.GeneratorClient.Stop(); err != nil {
		a.logger.Errorf("Error stopping generator client: %v", err)
	} else {
		a.logger.Infof("Generator client stopped successfully")
	}

	// Step 2: Complete processing the data that has already arrived
	a.logger.Infof("Step 2: Completing processing of remaining data...")
	if err := a.services.TelemetryService.Stop(ctx); err != nil {
		a.logger.Errorf("Error stopping telemetry service: %v", err)
	} else {
		a.logger.Infof("Telemetry service stopped successfully")
	}

	// Step 3: Stop HTTP server and other services
	a.logger.Infof("Step 3: Stopping HTTP server and other services...")
	wg := &sync.WaitGroup{}
	a.stopHttpServer(ctx, wg)
	a.closeServiceContext(wg)
	a.awaitShutdown(ctx, wg)

	// Cancel all context functions
	for _, cancelFunc := range a.cancelFuncs {
		cancelFunc()
	}
}

func (a *app) stopHttpServer(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go a.services.HttpServer.GracefulStop(ctx, wg)
}

func (a *app) closeServiceContext(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-a.ctx.Done():
			// If context was cancelled, continue
		case <-time.After(2 * time.Second):
			a.logger.Warnf("Service context closure timed out")
		}
	}()
}

func (a *app) awaitShutdown(ctx context.Context, wg *sync.WaitGroup) {
	gracefulStopChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(gracefulStopChan)
	}()
	select {
	case <-ctx.Done():
		a.logger.Warnf("%s stopped ungracefully", service.PrettyName)
	case <-gracefulStopChan:
		a.logger.Infof("%s stopped gracefully", service.PrettyName)
	}
}

func logVersion(logger log.Logger, serviceHome service.Home) {
	logger.Infof("%s service initialization started. PID: %d Home: %s",
		service.PrettyName, os.Getpid(), serviceHome.HomeDir())
}

func logSystemConfig(logger log.Logger, configService config.Service) {
	logger.Infof("Service configuration loaded successfully")
	logger.Infof("Service Type: %s", service.Type)
	logger.Infof("Service Service Id: %s", service.NewNodeInfo().GetServiceId())
	logger.Infof("Service Home Directory: %s", configService.GetHomeDir())
	logger.Infof("Service configuration loaded successfully")
}

func logNodeInfo(logger log.Logger, nodeInfo service.NodeInfo, configService config.Service) {
	logger.Infof("%s Node ID: %s", service.PrettyName, nodeInfo.GetNodeId())
	logger.Infof("%s Node IP: %s", service.PrettyName, configService.Get().Server.Host)
}

func (a *app) logInitCompleted() {
	startupDuration := time.Since(a.ctx.Value(ctxKeyStartTime).(time.Time))
	a.logger.Infof("%s service initialization completed in %.3f seconds",
		service.PrettyName, startupDuration.Seconds())
}

func newServices(ctx service.Context) *AppServices {
	tenantMiddlewares := middleware.NewMultiTenantMiddlewaresProvider()
	logger := ctx.LoggerFactory().(log.LoggerFactory).GetLogger("app")

	// Only if telemetry enabled - config can be changed during runtime, this feature might come handy
	// acting as a feature toggle
	var telemetryService telemetry.TelemetryService
	var telemetryHandler handler.TelemetryHandler
	var generatorClient *client.GeneratorClient

	if ctx.Config().Get().Telemetry.Enabled {
		// Initialize database connection
		db, err := sql.Open("postgres", ctx.Config().Get().Database.URL)
		if err != nil {
			logger.Fatalf("Failed to connect to database for telemetry: %v", err)
		}

		// Test database connection
		if err := db.PingContext(ctx); err != nil {
			logger.Warnf("Database ping failed for telemetry (continuing without telemetry): %v", err)
		} else {
			// Create storage components
			cache := storage.NewInMemoryCache()
			repository := storage.NewPostgreSQLRepository(db)

			// Configure hybrid store
			hybridConfig := storage.DefaultHybridStoreConfig()
			store := storage.NewHybridStore(cache, repository, hybridConfig, logger)

			// Create base telemetry service
			baseService := telemetry.NewTelemetryService(store, logger)

			// In case of know high load, enable queuing for telemetry requests
			// this is the last resort to avoid losing telemetry data in service level
			// in case its not enough, will start dropping requests.
			// Queuing is NOT recommended for normal loads, as it can introduce latency
			if ctx.Config().Get().Telemetry.Queue.Enabled {
				queueConfig := queue.QueueConfig{
					QueueSize:   ctx.Config().Get().Telemetry.Queue.QueueSize,
					Workers:     ctx.Config().Get().Telemetry.Queue.Workers,
					Timeout:     parseDuration(ctx.Config().Get().Telemetry.Queue.Timeout),
					EnableQueue: true,
				}
				telemetryService = telemetry.NewQueuedTelemetryService(baseService, queueConfig, logger)
				logger.Infof("Telemetry service initialized with request queuing enabled")
			} else {
				telemetryService = baseService
				logger.Infof("Telemetry service initialized without queuing (recommended for normal loads)")
			}

			telemetryHandler = handler.NewTelemetryHandler(ctx, telemetryService)

			// Since it configurable during runtime, it is possible to enable or disable telemetry ingestion
			if ctx.Config().Get().Telemetry.Ingestion.Enabled {
				clientConfig := client.GeneratorClientConfig{
					GeneratorURL:   ctx.Config().Get().Telemetry.Ingestion.GeneratorURL,
					PollInterval:   parseDuration(ctx.Config().Get().Telemetry.Ingestion.PollInterval),
					Timeout:        parseDuration(ctx.Config().Get().Telemetry.Ingestion.Timeout),
					MaxRetries:     ctx.Config().Get().Telemetry.Ingestion.MaxRetries,
					StartupDelay:   parseDuration(ctx.Config().Get().Telemetry.Ingestion.StartupDelay),
					ReadinessCheck: ctx.Config().Get().Telemetry.Ingestion.ReadinessCheck,
				}
				generatorClient = client.NewGeneratorClient(clientConfig, telemetryService, logger)
				logger.Infof("Generator client configured to poll %s every %s with %s startup delay",
					clientConfig.GeneratorURL, clientConfig.PollInterval, clientConfig.StartupDelay)
			}

			logger.Infof("Telemetry services initialized successfully")
		}
	}

	return &AppServices{
		HttpServer:       http.NewServer(ctx, tenantMiddlewares),
		SystemHandler:    handler.NewSystemHandler(ctx),
		TelemetryService: telemetryService,
		TelemetryHandler: telemetryHandler,
		GeneratorClient:  generatorClient,
	}
}

// Helper function to parse duration strings
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second // Default fallback
	}
	return d
}
