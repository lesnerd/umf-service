package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/http/middleware"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/service"
)

type RoutesInitializerFunc func(e *gin.Engine) []io.Closer

type ServeConfig struct {
	InitRoutes RoutesInitializerFunc
}

type Server interface {
	Listen(ctx service.Context) (*net.TCPAddr, error)
	Serve(config *ServeConfig) error
	GracefulStop(ctx context.Context, wg *sync.WaitGroup)
}

type server struct {
	server            *http.Server
	tcpListener       *net.TCPListener
	engine            *gin.Engine
	config            config.Service
	logger            log.Logger
	requestLogger     log.Logger
	closers           []io.Closer
	tenantMiddlewares middleware.MiddlewaresProvider
}

func NewServer(ctx service.Context, tenantMiddlewares middleware.MiddlewaresProvider) Server {
	logger := ctx.LoggerFactory().(log.LoggerFactory).GetLogger("http-server")
	serverInstance := server{
		config:            ctx.Config(),
		logger:            logger,
		requestLogger:     ctx.LoggerFactory().(log.LoggerFactory).GetRequestLogger(),
		closers:           []io.Closer{},
		tenantMiddlewares: tenantMiddlewares,
	}
	serverInstance.init(ctx)
	return &serverInstance
}

func (s *server) init(ctx service.Context) {
	serverMode := gin.ReleaseMode
	if s.config.Get().Logging.Level == "debug" {
		serverMode = gin.DebugMode
	}
	gin.SetMode(serverMode)
	s.engine = gin.New()

	// Add middleware
	s.engine.Use(s.tenantMiddlewares.ExtractTenantIdGinMiddleware())
	s.engine.Use(
		middleware.HandleTraceIdSetupFunc(s.logger),
		middleware.HandleGinLogsFunc(s.logger, s.requestLogger),
		middleware.HandleUnexpectedPanicRecoveryFunc(s.logger),
	)

	s.engine.UseRawPath = true
	s.engine.HandleMethodNotAllowed = true
	s.engine.ContextWithFallback = true
}

func (s *server) Listen(ctx service.Context) (*net.TCPAddr, error) {
	configuredPort := s.config.Get().Server.Port
	if configuredPort == "" || configuredPort == "0" {
		s.logger.Warnf("HTTP port is set to 0, an available port will be chosen automatically.")
		configuredPort = "0"
	}

	addr := s.config.Get().Server.Host + ":" + configuredPort
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: time.Minute,
	}
	s.closers = append(s.closers, s.server)

	// The following code is a copy of server.ListenAndServe()
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return nil, err
	}
	s.logger.Debugf("REST server listening on: %v", ln.Addr())
	s.tcpListener = ln.(*net.TCPListener)
	return s.tcpListener.Addr().(*net.TCPAddr), nil
}

func (s *server) Serve(config *ServeConfig) error {
	s.closers = append(s.closers, config.InitRoutes(s.engine)...)
	err := s.server.Serve(tcpKeepAliveListener{s.tcpListener})
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *server) GracefulStop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	if s.server == nil {
		return
	}

	s.logger.Infof("Stopping HTTP server...")

	// Shutdown the HTTP server with context timeout
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Warnf("Failed shutting down http server: %v", err)
	} else {
		s.logger.Infof("HTTP server stopped gracefully")
	}

	// Close all route handlers quickly
	for _, closer := range s.closers {
		if err := closer.Close(); err != nil {
			s.logger.Warnf("Failed when shutting down http route handler: %v", err)
		}
	}
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlivePeriod(3 * time.Minute)
	if err != nil {
		return nil, err
	}
	return tc, nil
}
