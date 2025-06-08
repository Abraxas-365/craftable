package logx

import (
	"io"
	"os"
	"strings"
)

var defaultLogger *Logger

func init() {
	defaultLogger = New()

	// Initialize from environment variables
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		if level, err := ParseLevel(logLevel); err == nil {
			defaultLogger.SetLevel(level)
		}
	}

	// Set format based on environment
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		switch strings.ToLower(format) {
		case "json":
			defaultLogger.SetFormat(FormatJSON)
		case "cloudwatch":
			defaultLogger.SetFormat(FormatCloudWatch)
		default:
			defaultLogger.SetFormat(FormatConsole)
		}
	}

	// Check for colored output (can be disabled with LOG_COLOR=false)
	if colorEnv := os.Getenv("LOG_COLOR"); colorEnv != "" {
		colored := strings.ToLower(colorEnv) != "false"
		defaultLogger.SetColored(colored)
	}

	// Check for caller info (can be disabled with LOG_CALLER=false)
	if callerEnv := os.Getenv("LOG_CALLER"); callerEnv != "" {
		showCaller := strings.ToLower(callerEnv) != "false"
		defaultLogger.SetShowCaller(showCaller)
	}
}

// SetLevel sets the global log level
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetPrefix sets the global log prefix
func SetPrefix(prefix string) {
	defaultLogger.SetPrefix(prefix)
}

// SetOutput sets the global output destination
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// SetShowCaller sets the global caller info display
func SetShowCaller(show bool) {
	defaultLogger.SetShowCaller(show)
}

// SetColored sets the global colored output
func SetColored(colored bool) {
	defaultLogger.SetColored(colored)
}

// SetFormat sets the global log format
func SetFormat(format OutputFormat) {
	defaultLogger.SetFormat(format)
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	return defaultLogger
}

// Global logging functions
func Trace(msg string, args ...any) {
	defaultLogger.Trace(msg, args...)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	defaultLogger.Fatal(msg, args...)
}

// DebugStruct logs a struct with full debug formatting globally
func DebugStruct(name string, value any) {
	defaultLogger.DebugStruct(name, value)
}

// TraceStruct logs a struct with full debug formatting at trace level globally
func TraceStruct(name string, value any) {
	defaultLogger.TraceStruct(name, value)
}

// IsLevelEnabled checks if a level is enabled globally
func IsLevelEnabled(level Level) bool {
	return defaultLogger.IsLevelEnabled(level)
}
