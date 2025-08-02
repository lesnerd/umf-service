package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ufm/internal/config"
	"github.com/ufm/internal/monitoring/tracing"
)

type Context interface {
	context.Context
	Home() Home
	Config() config.Service
	NodeInfo() NodeInfo
	Tracer() tracing.Tracer
	LoggerFactory() interface{}
	io.Closer
}

type serviceContext struct {
	cancelFunction context.CancelFunc
	innerCtx       context.Context
	home           Home
	configService  config.Service
	nodeInfo       NodeInfo
	tracer         tracing.Tracer
	loggerFactory  interface{}
}

func NewContext(
	ctx context.Context,
	home Home,
	configService config.Service,
	nodeInfo NodeInfo,
	tracer tracing.Tracer,
	loggerFactory interface{},
) Context {
	innerCtx, cancel := context.WithCancel(ctx)
	return &serviceContext{
		cancelFunction: cancel,
		innerCtx:       innerCtx,
		home:           home,
		configService:  configService,
		nodeInfo:       nodeInfo,
		tracer:         tracer,
		loggerFactory:  loggerFactory,
	}
}

func (c *serviceContext) Home() Home {
	return c.home
}

func (c *serviceContext) Config() config.Service {
	return c.configService
}

func (c *serviceContext) NodeInfo() NodeInfo {
	return c.nodeInfo
}

func (c *serviceContext) Tracer() tracing.Tracer {
	return c.tracer
}

func (c *serviceContext) LoggerFactory() interface{} {
	return c.loggerFactory
}

func (c *serviceContext) Deadline() (deadline time.Time, ok bool) {
	return c.innerCtx.Deadline()
}

func (c *serviceContext) Done() <-chan struct{} {
	return c.innerCtx.Done()
}

func (c *serviceContext) Err() error {
	return c.innerCtx.Err()
}

func (c *serviceContext) Close() error {
	c.cancelFunction()
	var collectedErrors []string
	if err := c.tracer.Close(); err != nil {
		// Log error without direct logger dependency
		fmt.Printf("Got error when closing the context: %+v\n", err)
		collectedErrors = append(collectedErrors, err.Error())
	}
	if len(collectedErrors) > 0 {
		return fmt.Errorf("Context close error(s): %s", strings.Join(collectedErrors, " | "))
	}
	return nil
}

func (c *serviceContext) Value(key interface{}) interface{} {
	return c.innerCtx.Value(key)
}

func SuppressCancellation(ctx context.Context) context.Context {
	return suppressCancellation{Context: ctx}
}

type suppressCancellation struct {
	context.Context
}

func (s suppressCancellation) Done() <-chan struct{} {
	return nil
}

func (s suppressCancellation) Err() error {
	return nil
}