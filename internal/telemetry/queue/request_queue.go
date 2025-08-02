package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry/models"
)

// RequestQueue manages high-volume API request queuing
type RequestQueue struct {
	requestChan chan *QueuedRequest
	workers     int
	workerPool  sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	logger      log.Logger
	handler     RequestHandler
	metrics     *QueueMetrics
	mu          sync.RWMutex
}

// QueuedRequest represents a queued API request
type QueuedRequest struct {
	RequestID    string
	RequestType  RequestType
	SwitchID     string
	MetricType   models.MetricType
	ResponseChan chan *QueuedResponse
	Timestamp    time.Time
}

// QueuedResponse represents the response to a queued request
type QueuedResponse struct {
	Data  interface{}
	Error error
}

// RequestType defines the type of API request
type RequestType int

const (
	GetMetricRequest RequestType = iota
	GetAllMetricsRequest
	ListAllSwitchesRequest
)

// RequestHandler defines the interface for handling queued requests
type RequestHandler interface {
	HandleGetMetric(switchID string, metricType models.MetricType) (interface{}, error)
	HandleGetAllMetrics(switchID string) (*models.TelemetryData, error)
	HandleListAllSwitches() map[string]*models.TelemetryData
}

// QueueMetrics tracks queue performance
type QueueMetrics struct {
	QueuedRequests    int64
	ProcessedRequests int64
	DroppedRequests   int64
	AverageWaitTime   time.Duration
	QueueDepth        int
	WorkerUtilization float64
}

// QueueConfig holds configuration for the request queue
type QueueConfig struct {
	QueueSize   int           // Size of the request buffer
	Workers     int           // Number of worker goroutines
	Timeout     time.Duration // Request timeout
	EnableQueue bool          // Whether to use queueing at all
}

// DefaultQueueConfig returns sensible defaults
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		QueueSize:   1000, // 1000 pending requests
		Workers:     10,   // 10 worker goroutines
		Timeout:     5 * time.Second,
		EnableQueue: false, // Disabled by default - only for extreme loads
	}
}

// NewRequestQueue creates a new request queue
func NewRequestQueue(config QueueConfig, handler RequestHandler, logger log.Logger) *RequestQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &RequestQueue{
		requestChan: make(chan *QueuedRequest, config.QueueSize),
		workers:     config.Workers,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		handler:     handler,
		metrics:     &QueueMetrics{},
	}
}

// Start initializes the queue workers
func (rq *RequestQueue) Start() error {
	rq.logger.Infof("Starting request queue with %d workers", rq.workers)

	for i := 0; i < rq.workers; i++ {
		rq.workerPool.Add(1)
		go rq.worker(i)
	}

	// Start metrics collection
	go rq.metricsCollector()

	return nil
}

// Stop gracefully shuts down the queue
func (rq *RequestQueue) Stop() error {
	rq.cancel()

	// Close request channel to signal workers to stop
	close(rq.requestChan)

	// Wait for all workers to finish
	rq.workerPool.Wait()

	rq.logger.Infof("Request queue stopped")
	return nil
}

// QueueGetMetric queues a GetMetric request
func (rq *RequestQueue) QueueGetMetric(requestID, switchID string, metricType models.MetricType) (interface{}, error) {
	responseChan := make(chan *QueuedResponse, 1)

	request := &QueuedRequest{
		RequestID:    requestID,
		RequestType:  GetMetricRequest,
		SwitchID:     switchID,
		MetricType:   metricType,
		ResponseChan: responseChan,
		Timestamp:    time.Now(),
	}

	// Try to queue the request (non-blocking)
	select {
	case rq.requestChan <- request:
		rq.incrementQueuedRequests()
	default:
		// Queue is full, drop request
		rq.incrementDroppedRequests()
		return nil, fmt.Errorf("request queue is full, dropping request")
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return response.Data, response.Error
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("request timeout")
	case <-rq.ctx.Done():
		return nil, fmt.Errorf("queue is shutting down")
	}
}

// QueueGetAllMetrics queues a GetAllMetrics request
func (rq *RequestQueue) QueueGetAllMetrics(requestID, switchID string) (*models.TelemetryData, error) {
	responseChan := make(chan *QueuedResponse, 1)

	request := &QueuedRequest{
		RequestID:    requestID,
		RequestType:  GetAllMetricsRequest,
		SwitchID:     switchID,
		ResponseChan: responseChan,
		Timestamp:    time.Now(),
	}

	select {
	case rq.requestChan <- request:
		rq.incrementQueuedRequests()
	default:
		rq.incrementDroppedRequests()
		return nil, fmt.Errorf("request queue is full, dropping request")
	}

	select {
	case response := <-responseChan:
		if data, ok := response.Data.(*models.TelemetryData); ok {
			return data, response.Error
		}
		return nil, response.Error
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("request timeout")
	case <-rq.ctx.Done():
		return nil, fmt.Errorf("queue is shutting down")
	}
}

// worker processes queued requests
func (rq *RequestQueue) worker(workerID int) {
	defer rq.workerPool.Done()

	rq.logger.Debugf("Request queue worker %d started", workerID)

	for {
		select {
		case request, ok := <-rq.requestChan:
			if !ok {
				// Channel closed, worker should exit
				rq.logger.Debugf("Request queue worker %d stopping", workerID)
				return
			}

			rq.processRequest(request)
			rq.incrementProcessedRequests()

		case <-rq.ctx.Done():
			rq.logger.Debugf("Request queue worker %d stopping due to context cancellation", workerID)
			return
		}
	}
}

// processRequest handles a single queued request
func (rq *RequestQueue) processRequest(request *QueuedRequest) {
	defer func() {
		if r := recover(); r != nil {
			rq.logger.Errorf("Panic in request queue worker: %v", r)
			request.ResponseChan <- &QueuedResponse{
				Error: fmt.Errorf("internal server error"),
			}
		}
	}()

	var response *QueuedResponse

	switch request.RequestType {
	case GetMetricRequest:
		data, err := rq.handler.HandleGetMetric(request.SwitchID, request.MetricType)
		response = &QueuedResponse{Data: data, Error: err}

	case GetAllMetricsRequest:
		data, err := rq.handler.HandleGetAllMetrics(request.SwitchID)
		response = &QueuedResponse{Data: data, Error: err}

	case ListAllSwitchesRequest:
		data := rq.handler.HandleListAllSwitches()
		response = &QueuedResponse{Data: data, Error: nil}

	default:
		response = &QueuedResponse{
			Error: fmt.Errorf("unknown request type: %d", request.RequestType),
		}
	}

	// Send response back
	select {
	case request.ResponseChan <- response:
		// Response sent successfully
	default:
		// Response channel is full or closed, log warning
		rq.logger.Warnf("Failed to send response for request %s", request.RequestID)
	}
}

// metricsCollector periodically updates queue metrics
func (rq *RequestQueue) metricsCollector() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rq.updateMetrics()
		case <-rq.ctx.Done():
			return
		}
	}
}

// updateMetrics calculates current queue metrics
func (rq *RequestQueue) updateMetrics() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.metrics.QueueDepth = len(rq.requestChan)

	// Calculate worker utilization (simplified)
	if rq.metrics.QueueDepth > 0 {
		rq.metrics.WorkerUtilization = float64(rq.metrics.QueueDepth) / float64(rq.workers) * 100
		if rq.metrics.WorkerUtilization > 100 {
			rq.metrics.WorkerUtilization = 100
		}
	} else {
		rq.metrics.WorkerUtilization = 0
	}
}

// GetMetrics returns current queue metrics
func (rq *RequestQueue) GetMetrics() QueueMetrics {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	return *rq.metrics
}

// Helper methods for metrics
func (rq *RequestQueue) incrementQueuedRequests() {
	rq.mu.Lock()
	rq.metrics.QueuedRequests++
	rq.mu.Unlock()
}

func (rq *RequestQueue) incrementProcessedRequests() {
	rq.mu.Lock()
	rq.metrics.ProcessedRequests++
	rq.mu.Unlock()
}

func (rq *RequestQueue) incrementDroppedRequests() {
	rq.mu.Lock()
	rq.metrics.DroppedRequests++
	rq.mu.Unlock()
}
