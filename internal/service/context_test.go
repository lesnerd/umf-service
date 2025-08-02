package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ufm/internal/config"
	"github.com/ufm/internal/monitoring/tracing"
)

// Mock implementations
type mockConfigService struct {
	mock.Mock
}

func (m *mockConfigService) Get() config.Config {
	args := m.Called()
	return args.Get(0).(config.Config)
}

func (m *mockConfigService) AddUpdateListener(listener config.UpdateListener) {
	m.Called(listener)
}

func (m *mockConfigService) IsMultiTenant() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockConfigService) GetHomeDir() string {
	args := m.Called()
	return args.String(0)
}

type mockTracer struct {
	mock.Mock
}

func (m *mockTracer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTracer) StartSpanFromContext(ctx context.Context, operationName string) (context.Context, tracing.SpanCloseFunction) {
	args := m.Called(ctx, operationName)
	return args.Get(0).(context.Context), args.Get(1).(tracing.SpanCloseFunction)
}

type mockLoggerFactory struct {
	mock.Mock
}

func TestNewContext(t *testing.T) {
	// Create mock dependencies
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	// Set up expectations
	mockHome.On("HomeDir").Return("/tmp/test-home")
	mockConfig.On("GetHomeDir").Return("/tmp/test-home")
	mockNodeInfo.On("GetNodeId").Return("test-node")
	mockNodeInfo.On("GetServiceId").Return("test-service")

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	assert.NotNil(t, serviceCtx)
	assert.Implements(t, (*Context)(nil), serviceCtx)
	assert.Implements(t, (*context.Context)(nil), serviceCtx)

	// Test interface methods
	assert.Equal(t, mockHome, serviceCtx.Home())
	assert.Equal(t, mockConfig, serviceCtx.Config())
	assert.Equal(t, mockNodeInfo, serviceCtx.NodeInfo())
	assert.Equal(t, mockTracer, serviceCtx.Tracer())
	assert.Equal(t, mockLoggerFactory, serviceCtx.LoggerFactory())
}

func TestServiceContext_Home(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	result := serviceCtx.Home()
	assert.Equal(t, mockHome, result)
}

func TestServiceContext_Config(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	result := serviceCtx.Config()
	assert.Equal(t, mockConfig, result)
}

func TestServiceContext_NodeInfo(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	result := serviceCtx.NodeInfo()
	assert.Equal(t, mockNodeInfo, result)
}

func TestServiceContext_Tracer(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	result := serviceCtx.Tracer()
	assert.Equal(t, mockTracer, result)
}

func TestServiceContext_LoggerFactory(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	result := serviceCtx.LoggerFactory()
	assert.Equal(t, mockLoggerFactory, result)
}

func TestServiceContext_ContextMethods(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Test Deadline
	deadline, ok := serviceCtx.Deadline()
	assert.False(t, ok) // Background context has no deadline
	assert.True(t, deadline.IsZero())

	// Test Done
	done := serviceCtx.Done()
	assert.NotNil(t, done)

	// Test Err
	err := serviceCtx.Err()
	assert.NoError(t, err)

	// Test Value
	value := serviceCtx.Value("test-key")
	assert.Nil(t, value)
}

func TestServiceContext_Close(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	// Set up tracer expectation
	mockTracer.On("Close").Return(nil)

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Test Close
	err := serviceCtx.Close()
	assert.NoError(t, err)

	// Verify tracer was called
	mockTracer.AssertExpectations(t)
}

func TestServiceContext_CloseWithTracerError(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	// Set up tracer expectation to return error
	mockTracer.On("Close").Return(assert.AnError)

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Test Close with error
	err := serviceCtx.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Context close error(s)")

	// Verify tracer was called
	mockTracer.AssertExpectations(t)
}

func TestServiceContext_Cancellation(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	// Set up tracer expectation
	mockTracer.On("Close").Return(nil)

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Close the context
	err := serviceCtx.Close()
	assert.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-serviceCtx.Done():
		// Context is cancelled as expected
	default:
		t.Error("Context should be cancelled after Close()")
	}

	assert.Error(t, serviceCtx.Err())
	assert.Equal(t, context.Canceled, serviceCtx.Err())
}

func TestServiceContext_WithTimeout(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	// Set up tracer expectation
	mockTracer.On("Close").Return(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Wait for timeout
	select {
	case <-serviceCtx.Done():
		// Context timed out as expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}

	assert.Error(t, serviceCtx.Err())
	assert.Equal(t, context.DeadlineExceeded, serviceCtx.Err())
}

func TestSuppressCancellation(t *testing.T) {
	ctx := context.Background()
	suppressedCtx := SuppressCancellation(ctx)

	assert.NotNil(t, suppressedCtx)

	// Test that Done() returns nil
	done := suppressedCtx.Done()
	assert.Nil(t, done)

	// Test that Err() returns nil
	err := suppressedCtx.Err()
	assert.NoError(t, err)

	// Test that other context methods still work
	deadline, ok := suppressedCtx.Deadline()
	assert.False(t, ok)
	assert.True(t, deadline.IsZero())

	value := suppressedCtx.Value("test-key")
	assert.Nil(t, value)
}

func TestServiceContext_InterfaceCompliance(t *testing.T) {
	mockHome := &mockHome{}
	mockConfig := &mockConfigService{}
	mockNodeInfo := &mockNodeInfo{}
	mockTracer := &mockTracer{}
	mockLoggerFactory := &mockLoggerFactory{}

	ctx := context.Background()
	serviceCtx := NewContext(ctx, mockHome, mockConfig, mockNodeInfo, mockTracer, mockLoggerFactory)

	// Test that the returned object implements all required interfaces
	var _ Context = serviceCtx
	var _ context.Context = serviceCtx
}

// Mock implementations for testing
type mockHome struct {
	mock.Mock
}

func (m *mockHome) HomeDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockHome) LogDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockHome) DataDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockHome) ConfigDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockHome) SystemConfigFile() string {
	args := m.Called()
	return args.String(0)
}

type mockNodeInfo struct {
	mock.Mock
}

func (m *mockNodeInfo) GetNodeId() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockNodeInfo) GetServiceId() string {
	args := m.Called()
	return args.String(0)
}
