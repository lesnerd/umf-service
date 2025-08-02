package log

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type LoggerFactory interface {
	GetLogger(name string) Logger
	GetRootLogger() Logger
	GetRequestLogger() Logger
}

type LoggingConfig struct {
	Level    string
	Format   string
	Console  bool
	FilePath string
}

type loggerFactory struct {
	rootLogger    Logger
	requestLogger Logger
	config        LoggingConfig
}

func NewLoggerFactory(ctx context.Context, logger Logger, loggingConfig LoggingConfig) LoggerFactory {
	factory := &loggerFactory{
		rootLogger: logger,
		config:     loggingConfig,
	}

	// Create request logger
	factory.requestLogger = factory.createRequestLogger()

	return factory
}

func (f *loggerFactory) GetLogger(name string) Logger {
	return &namedLogger{
		logger: f.rootLogger,
		name:   name,
	}
}

func (f *loggerFactory) GetRootLogger() Logger {
	return f.rootLogger
}

func (f *loggerFactory) GetRequestLogger() Logger {
	return f.requestLogger
}

func (f *loggerFactory) createRequestLogger() Logger {
	logger := logrus.New()

	// Configure output
	if f.config.Console {
		logger.SetOutput(os.Stdout)
	} else if f.config.FilePath != "" {
		if file, err := os.OpenFile(f.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(file)
		}
	}

	// Configure format
	if f.config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else if f.config.Format == "pretty" {
		logger.SetFormatter(NewPrettyFormatter(true, true))
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Configure level
	level, err := logrus.ParseLevel(f.config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	return &logrusLogger{logger: logger}
}

type namedLogger struct {
	logger Logger
	name   string
}

func (l *namedLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) Tracef(format string, args ...interface{}) {
	l.logger.Tracef("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf("[%s] "+format, append([]interface{}{l.name}, args...)...)
}

func (l *namedLogger) WithField(key string, value interface{}) Logger {
	return l.logger.WithField(key, value)
}

func (l *namedLogger) WithFields(fields map[string]interface{}) Logger {
	return l.logger.WithFields(fields)
}

func (l *namedLogger) Writer() io.Writer {
	return l.logger.Writer()
}
