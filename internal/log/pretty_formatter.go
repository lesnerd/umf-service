package log

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// PrettyFormatter provides a clean, readable log format with trace ID support
type PrettyFormatter struct {
	// Enable colors in output
	EnableColors bool
	// Show file and line information when available
	ShowFileLine bool
}

// NewPrettyFormatter creates a new pretty formatter
func NewPrettyFormatter(enableColors, showFileLine bool) *PrettyFormatter {
	return &PrettyFormatter{
		EnableColors: enableColors,
		ShowFileLine: showFileLine,
	}
}

// Format formats a log entry
func (f *PrettyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b strings.Builder

	// Timestamp
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	b.WriteString(timestamp)
	b.WriteString(" | ")

	// Log level with color
	level := strings.ToUpper(entry.Level.String())
	// Map WARNING to WARN for consistency
	if level == "WARNING" {
		level = "WARN"
	}
	if f.EnableColors {
		level = f.colorizeLevel(level, entry.Level)
	}
	b.WriteString(fmt.Sprintf("%-5s", level))
	b.WriteString(" | ")

	// Trace ID
	traceID := f.extractTraceID(entry)
	if traceID != "" {
		b.WriteString(fmt.Sprintf("[%s] | ", traceID))
	}

	// Logger name (from fields or default)
	loggerName := f.extractLoggerName(entry)
	if loggerName != "" {
		b.WriteString(loggerName)
		b.WriteString(" | ")
	}

	// File and line information
	if f.ShowFileLine {
		if caller := f.extractCaller(entry); caller != "" {
			b.WriteString(caller)
			b.WriteString(" | ")
		}
	}

	// Message
	b.WriteString(entry.Message)

	// Additional fields (excluding trace_id, logger_name, file, line)
	additionalFields := f.extractAdditionalFields(entry)
	if len(additionalFields) > 0 {
		b.WriteString(" | ")
		b.WriteString(additionalFields)
	}

	b.WriteString("\n")
	return []byte(b.String()), nil
}

// colorizeLevel adds ANSI color codes to log levels
func (f *PrettyFormatter) colorizeLevel(level string, logLevel logrus.Level) string {
	if !f.EnableColors {
		return level
	}

	const (
		red    = "\033[31m"
		yellow = "\033[33m"
		blue   = "\033[34m"
		green  = "\033[32m"
		cyan   = "\033[36m"
		reset  = "\033[0m"
	)

	switch logLevel {
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return red + level + reset
	case logrus.WarnLevel:
		return yellow + level + reset
	case logrus.InfoLevel:
		return blue + level + reset
	case logrus.DebugLevel:
		return green + level + reset
	case logrus.TraceLevel:
		return cyan + level + reset
	default:
		return level
	}
}

// extractTraceID extracts trace ID from entry fields
func (f *PrettyFormatter) extractTraceID(entry *logrus.Entry) string {
	// Check for trace_id field
	if traceID, exists := entry.Data["trace_id"]; exists {
		if str, ok := traceID.(string); ok && str != "" {
			// Shorten UUID to 12 characters for readability only if longer
			if len(str) > 12 {
				return str[:12]
			}
			return str
		}
	}

	// Check for traceId field (alternative naming)
	if traceID, exists := entry.Data["traceId"]; exists {
		if str, ok := traceID.(string); ok && str != "" {
			// Shorten UUID to 12 characters for readability only if longer
			if len(str) > 12 {
				return str[:12]
			}
			return str
		}
	}

	return ""
}

// extractLoggerName extracts logger name from entry fields or message
func (f *PrettyFormatter) extractLoggerName(entry *logrus.Entry) string {
	// Check for logger_name field
	if loggerName, exists := entry.Data["logger_name"]; exists {
		if str, ok := loggerName.(string); ok && str != "" {
			return str
		}
	}

	// Check for component field
	if component, exists := entry.Data["component"]; exists {
		if str, ok := component.(string); ok && str != "" {
			return str
		}
	}

	// Try to extract from message if it starts with [name]
	msg := entry.Message
	if strings.HasPrefix(msg, "[") {
		end := strings.Index(msg, "]")
		if end > 1 {
			name := msg[1:end]
			// Remove the [name] prefix from message
			entry.Message = strings.TrimSpace(msg[end+1:])
			return name
		}
	}

	return ""
}

// extractCaller extracts file and line information
func (f *PrettyFormatter) extractCaller(entry *logrus.Entry) string {
	if entry.Caller != nil {
		file := entry.Caller.File
		line := entry.Caller.Line

		// Extract just the filename without path
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			file = file[idx+1:]
		}

		return fmt.Sprintf("%s:%d", file, line)
	}

	// Check for file and line in fields
	if file, exists := entry.Data["file"]; exists {
		if line, exists := entry.Data["line"]; exists {
			if fileStr, ok := file.(string); ok {
				if lineNum, ok := line.(int); ok {
					// Extract just the filename without path
					if idx := strings.LastIndex(fileStr, "/"); idx != -1 {
						fileStr = fileStr[idx+1:]
					}
					return fmt.Sprintf("%s:%d", fileStr, lineNum)
				}
			}
		}
	}

	return ""
}

// extractAdditionalFields extracts additional fields for display
func (f *PrettyFormatter) extractAdditionalFields(entry *logrus.Entry) string {
	var fields []string

	// Fields to exclude from additional display
	excludedFields := map[string]bool{
		"trace_id":    true,
		"traceId":     true,
		"logger_name": true,
		"component":   true,
		"file":        true,
		"line":        true,
		"timestamp":   true,
		"level":       true,
		"msg":         true,
	}

	for key, value := range entry.Data {
		if !excludedFields[key] {
			fields = append(fields, fmt.Sprintf("%s=%v", key, value))
		}
	}

	return strings.Join(fields, ", ")
}
