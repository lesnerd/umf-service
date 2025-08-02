package tracing

import (
	"context"
	"io"

	"github.com/ufm/internal/log"
	"github.com/google/uuid"
)

const Id = "tracing.traceId"

type SpanCloseFunction func()

type Tracer interface {
	StartSpanFromContext(ctx context.Context, operationName string) (context.Context, SpanCloseFunction)
	io.Closer
}

type tracer struct {
	serviceName string
	logger      log.Logger
}

func NewTracer(serviceName string, logger log.Logger) Tracer {
	return &tracer{
		serviceName: serviceName,
		logger:      logger,
	}
}

func (t *tracer) Close() error {
	// Nothing to close in this simple implementation
	return nil
}

// StartSpanFromContext starts a root (parent-less) span with `operationName`.
// The return value is a context built around the created Span,
// which is a child of the provided context.
func (t *tracer) StartSpanFromContext(ctx context.Context, operationName string) (context.Context, SpanCloseFunction) {
	traceId := uuid.New().String()
	newContext := context.WithValue(ctx, Id, traceId)
	
	t.logger.Debugf("Starting span: %s with trace ID: %s", operationName, traceId)
	
	return newContext, func() {
		t.logger.Debugf("Finishing span: %s with trace ID: %s", operationName, traceId)
	}
}

func ExtractTraceId(ctx context.Context) string {
	str, isString := ctx.Value(Id).(string)
	if isString {
		return str
	}
	return ""
}