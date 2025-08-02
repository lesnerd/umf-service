package log

import (
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPrettyFormatter_Format(t *testing.T) {
	formatter := NewPrettyFormatter(true, true)

	tests := []struct {
		name     string
		entry    *logrus.Entry
		expected string
	}{
		{
			name: "basic info log",
			entry: &logrus.Entry{
				Level:   logrus.InfoLevel,
				Message: "Server started successfully",
				Time:    time.Date(2025, 8, 2, 7, 53, 41, 689000000, time.UTC),
				Data: map[string]interface{}{
					"logger_name": "app",
				},
			},
			expected: "2025-08-02 07:53:41 | INFO | app | Server started successfully\n",
		},
		{
			name: "warning log with trace ID",
			entry: &logrus.Entry{
				Level:   logrus.WarnLevel,
				Message: "Failed to initialize tenant registry client: context deadline exceeded",
				Time:    time.Date(2025, 8, 2, 7, 53, 41, 689000000, time.UTC),
				Data: map[string]interface{}{
					"logger_name": "tenant-registry-client",
					"trace_id":    "1ced733cdf0b6de3",
				},
			},
			expected: "2025-08-02 07:53:41 | WARN | [1ced733cdf0b] | tenant-registry-client | Failed to initialize tenant registry client: context deadline exceeded\n",
		},
		{
			name: "error log with file and line",
			entry: &logrus.Entry{
				Level:   logrus.ErrorLevel,
				Message: "Database connection failed",
				Time:    time.Date(2025, 8, 2, 7, 53, 41, 689000000, time.UTC),
				Data: map[string]interface{}{
					"logger_name": "database",
					"trace_id":    "1ced733cdf0b6de3",
					"file":        "database.go",
					"line":        42,
				},
			},
			expected: "2025-08-02 07:53:41 | ERROR | [1ced733cdf0b] | database | database.go:42 | Database connection failed\n",
		},
		{
			name: "http request log",
			entry: &logrus.Entry{
				Level:   logrus.InfoLevel,
				Message: "GET /api/v1/telemetry 200 1.2ms",
				Time:    time.Date(2025, 8, 2, 7, 53, 41, 689000000, time.UTC),
				Data: map[string]interface{}{
					"logger_name": "http-server",
					"trace_id":    "1ced733cdf0b6de3",
					"method":      "GET",
					"path":        "/api/v1/telemetry",
					"status":      200,
					"latency":     "1.2ms",
				},
			},
			expected: "2025-08-02 07:53:41 | INFO | [1ced733cdf0b] | http-server | GET /api/v1/telemetry 200 1.2ms | method=GET, path=/api/v1/telemetry, status=200, latency=1.2ms\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.entry)
			assert.NoError(t, err)

			// Remove color codes for comparison
			resultStr := string(result)
			resultStr = removeColorCodes(resultStr)

			// For the http request log, check that all expected fields are present
			// but don't enforce specific order since map iteration order is not guaranteed
			if tt.name == "http request log" {
				assert.Contains(t, resultStr, "method=GET")
				assert.Contains(t, resultStr, "path=/api/v1/telemetry")
				assert.Contains(t, resultStr, "status=200")
				assert.Contains(t, resultStr, "latency=1.2ms")
				assert.Contains(t, resultStr, "GET /api/v1/telemetry 200 1.2ms")
			} else {
				assert.Equal(t, tt.expected, resultStr)
			}
		})
	}
}

func TestPrettyFormatter_ExtractTraceID(t *testing.T) {
	formatter := NewPrettyFormatter(true, true)

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{
			name: "trace_id field",
			data: map[string]interface{}{
				"trace_id": "1ced733cdf0b6de3",
			},
			expected: "1ced733cdf0b",
		},
		{
			name: "traceId field",
			data: map[string]interface{}{
				"traceId": "1ced733cdf0b6de3",
			},
			expected: "1ced733cdf0b",
		},
		{
			name: "long trace ID should be shortened",
			data: map[string]interface{}{
				"trace_id": "1ced733cdf0b6de3abcd1234",
			},
			expected: "1ced733cdf0b",
		},
		{
			name:     "no trace ID",
			data:     map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &logrus.Entry{Data: tt.data}
			result := formatter.extractTraceID(entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrettyFormatter_ExtractLoggerName(t *testing.T) {
	formatter := NewPrettyFormatter(true, true)

	tests := []struct {
		name     string
		entry    *logrus.Entry
		expected string
	}{
		{
			name: "logger_name field",
			entry: &logrus.Entry{
				Message: "test message",
				Data: map[string]interface{}{
					"logger_name": "test-logger",
				},
			},
			expected: "test-logger",
		},
		{
			name: "component field",
			entry: &logrus.Entry{
				Message: "test message",
				Data: map[string]interface{}{
					"component": "test-component",
				},
			},
			expected: "test-component",
		},
		{
			name: "extract from message prefix",
			entry: &logrus.Entry{
				Message: "[test-logger] test message",
			},
			expected: "test-logger",
		},
		{
			name: "no logger name",
			entry: &logrus.Entry{
				Message: "test message",
				Data:    map[string]interface{}{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.extractLoggerName(tt.entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to remove ANSI color codes
func removeColorCodes(s string) string {
	// Simple regex to remove ANSI color codes
	// This is a basic implementation - in production you might want a more robust solution
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}
