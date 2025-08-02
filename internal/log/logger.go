package log

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	Writer() io.Writer
}

type logrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

func NewDefaultLogger() Logger {
	logger := logrus.New()
	logger.SetFormatter(NewPrettyFormatter(true, true))
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(os.Stdout)

	return &logrusLogger{logger: logger}
}

func NewLoggerWithConfig(level, format string, output io.Writer) Logger {
	logger := logrus.New()

	// Set format
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else if format == "pretty" {
		logger.SetFormatter(NewPrettyFormatter(true, true))
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Set level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set output
	if output != nil {
		logger.SetOutput(output)
	} else {
		logger.SetOutput(os.Stdout)
	}

	return &logrusLogger{logger: logger}
}

func (l *logrusLogger) Infof(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Infof(format, args...)
	} else {
		l.logger.Infof(format, args...)
	}
}

func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Debugf(format, args...)
	} else {
		l.logger.Debugf(format, args...)
	}
}

func (l *logrusLogger) Tracef(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Tracef(format, args...)
	} else {
		l.logger.Tracef(format, args...)
	}
}

func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Warnf(format, args...)
	} else {
		l.logger.Warnf(format, args...)
	}
}

func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Errorf(format, args...)
	} else {
		l.logger.Errorf(format, args...)
	}
}

func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	if l.entry != nil {
		l.entry.Fatalf(format, args...)
	} else {
		l.logger.Fatalf(format, args...)
	}
}

func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	var entry *logrus.Entry
	if l.entry != nil {
		entry = l.entry.WithField(key, value)
	} else {
		entry = l.logger.WithField(key, value)
	}
	return &logrusLogger{logger: l.logger, entry: entry}
}

func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	var entry *logrus.Entry
	if l.entry != nil {
		entry = l.entry.WithFields(fields)
	} else {
		entry = l.logger.WithFields(fields)
	}
	return &logrusLogger{logger: l.logger, entry: entry}
}

func (l *logrusLogger) Writer() io.Writer {
	return l.logger.Writer()
}

var DefaultLogger = NewLoggerWithConfig("info", "pretty", os.Stdout)
