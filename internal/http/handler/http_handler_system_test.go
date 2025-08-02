package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/monitoring/tracing"
	"github.com/ufm/internal/service"
)

// Mock implementations
type mockContext struct {
	mock.Mock
}

func (m *mockContext) Home() service.Home {
	args := m.Called()
	return args.Get(0).(service.Home)
}

func (m *mockContext) Config() config.Service {
	args := m.Called()
	return args.Get(0).(config.Service)
}

func (m *mockContext) NodeInfo() service.NodeInfo {
	args := m.Called()
	return args.Get(0).(service.NodeInfo)
}

func (m *mockContext) Tracer() tracing.Tracer {
	args := m.Called()
	return args.Get(0).(tracing.Tracer)
}

func (m *mockContext) LoggerFactory() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *mockContext) Deadline() (time.Time, bool) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Bool(1)
}

func (m *mockContext) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(<-chan struct{})
}

func (m *mockContext) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockContext) Value(key interface{}) interface{} {
	args := m.Called(key)
	return args.Get(0)
}

func (m *mockContext) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockLoggerFactory struct {
	mock.Mock
}

func (m *mockLoggerFactory) GetLogger(name string) log.Logger {
	args := m.Called(name)
	return args.Get(0).(log.Logger)
}

func (m *mockLoggerFactory) GetRootLogger() log.Logger {
	args := m.Called()
	return args.Get(0).(log.Logger)
}

func (m *mockLoggerFactory) GetRequestLogger() log.Logger {
	args := m.Called()
	return args.Get(0).(log.Logger)
}

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) Fatalf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) Tracef(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockLogger) WithField(key string, value interface{}) log.Logger {
	args := m.Called(key, value)
	return args.Get(0).(log.Logger)
}

func (m *mockLogger) WithFields(fields map[string]interface{}) log.Logger {
	args := m.Called(fields)
	return args.Get(0).(log.Logger)
}

func (m *mockLogger) Writer() io.Writer {
	args := m.Called()
	return args.Get(0).(io.Writer)
}

func setupTestRouter(handler SystemHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add routes
	router.GET("/ping", handler.Ping)
	router.GET("/health", handler.Health)
	router.GET("/readiness", handler.Readiness)
	router.GET("/version", handler.Version)

	return router
}

func TestNewSystemHandler(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)

	assert.NotNil(t, handler)
	assert.Implements(t, (*SystemHandler)(nil), handler)

	mockCtx.AssertExpectations(t)
	mockLoggerFactory.AssertExpectations(t)
}

func TestSystemHandler_Ping(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	// Create request
	req, err := http.NewRequest("GET", "/ping", nil)
	assert.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, service.PrettyName, response["service"])
	assert.Contains(t, response, "timestamp")

	// Verify timestamp is valid
	timestampStr, ok := response["timestamp"].(string)
	assert.True(t, ok)
	_, err = time.Parse(time.RFC3339, timestampStr)
	assert.NoError(t, err)
}

func TestSystemHandler_Health(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, service.PrettyName, response["service"])
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "checks")

	// Verify checks
	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "ok", checks["database"])
}

func TestSystemHandler_Readiness(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	// Create request
	req, err := http.NewRequest("GET", "/readiness", nil)
	assert.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "ready", response["status"])
	assert.Equal(t, service.PrettyName, response["service"])
	assert.Contains(t, response, "timestamp")
}

func TestSystemHandler_Version(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	// Create request
	req, err := http.NewRequest("GET", "/version", nil)
	assert.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, service.PrettyName, response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "dev", response["build"])
	assert.Equal(t, "unknown", response["commit"])
}

func TestSystemHandler_InterfaceCompliance(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)

	// Test that the returned object implements the SystemHandler interface
	var _ SystemHandler = handler
}

func TestSystemHandler_ResponseHeaders(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	// Test ping endpoint headers
	req, err := http.NewRequest("GET", "/ping", nil)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that response has proper content type
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSystemHandler_AllEndpoints(t *testing.T) {
	mockCtx := &mockContext{}
	mockLoggerFactory := &mockLoggerFactory{}
	mockLogger := &mockLogger{}

	// Set up expectations
	mockCtx.On("LoggerFactory").Return(mockLoggerFactory)
	mockLoggerFactory.On("GetLogger", "system-handler").Return(mockLogger)

	handler := NewSystemHandler(mockCtx)
	router := setupTestRouter(handler)

	endpoints := []string{"/ping", "/health", "/readiness", "/version"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req, err := http.NewRequest("GET", endpoint, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// All endpoints should return 200 OK
			assert.Equal(t, http.StatusOK, w.Code)

			// All responses should be valid JSON
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Most responses should have a timestamp (except version)
			if endpoint != "/version" {
				assert.Contains(t, response, "timestamp")
			}
		})
	}
}
