package logx

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// OutputFormat defines the log output format
type OutputFormat string

const (
	FormatConsole    OutputFormat = "console"
	FormatCloudWatch OutputFormat = "cloudwatch"
	FormatJSON       OutputFormat = "json"
)

// Logger represents a logger instance
type Logger struct {
	level          Level
	out            io.Writer
	prefix         string
	showCaller     bool
	colored        bool
	format         OutputFormat
	debugFormatter *DebugFormatter
	cloudFormatter *CloudWatchFormatter
}

// New creates a new logger with default settings
func New() *Logger {
	return &Logger{
		level:          InfoLevel,
		out:            os.Stdout,
		prefix:         "",
		showCaller:     true,
		colored:        true,
		format:         FormatConsole,
		debugFormatter: NewDebugFormatter(),
		cloudFormatter: NewCloudWatchFormatter(false),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetOutput sets the output destination
func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
}

// SetPrefix sets a prefix for all log messages
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// SetShowCaller enables or disables showing caller information
func (l *Logger) SetShowCaller(show bool) {
	l.showCaller = show
}

// SetColored enables or disables colored output
func (l *Logger) SetColored(colored bool) {
	l.colored = colored
}

// SetFormat sets the output format
func (l *Logger) SetFormat(format OutputFormat) {
	l.format = format
	// Disable colors for CloudWatch and JSON formats
	if format == FormatCloudWatch || format == FormatJSON {
		l.colored = false
	}
	// Update CloudWatch formatter for JSON mode
	if format == FormatJSON {
		l.cloudFormatter = NewCloudWatchFormatter(true)
	}
}

// IsLevelEnabled checks if a level is enabled
func (l *Logger) IsLevelEnabled(level Level) bool {
	return level >= l.level
}

// findCaller finds the first caller outside of the logx package
func (l *Logger) findCaller() string {
	if !l.showCaller {
		return ""
	}

	// Start from frame 1 (skip this function itself)
	for i := 1; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		filename := filepath.Base(file)

		// Skip internal Go runtime files
		if strings.HasPrefix(filename, "proc.go") ||
			strings.HasPrefix(filename, "runtime") ||
			strings.HasPrefix(filename, "asm_") {
			continue
		}

		// Skip logx package files (more robust check)
		if strings.Contains(file, "logx") &&
			(strings.HasSuffix(file, "/logger.go") ||
				strings.HasSuffix(file, "/global.go") ||
				strings.HasSuffix(file, "/formatter.go") ||
				strings.HasSuffix(file, "/level.go")) {
			continue
		}

		// This should be the actual caller
		return fmt.Sprintf(" %s:%d", filename, line)
	}

	return ""
}

// logInternal is the core logging function
func (l *Logger) logInternal(level Level, formatArgs bool, msg string, args ...any) {
	if !l.IsLevelEnabled(level) {
		return
	}

	// Handle different output formats
	switch l.format {
	case FormatJSON:
		l.logJSON(level, msg, args...)
	case FormatCloudWatch:
		l.logCloudWatch(level, formatArgs, msg, args...)
	default:
		l.logConsole(level, formatArgs, msg, args...)
	}
}

// logJSON outputs structured JSON logs
func (l *Logger) logJSON(level Level, msg string, args ...any) {
	logEntry := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level.String(),
		"message":   fmt.Sprintf(msg, args...),
	}

	if l.prefix != "" {
		logEntry["prefix"] = l.prefix
	}

	if l.showCaller {
		caller := l.findCaller()
		if caller != "" {
			logEntry["caller"] = strings.TrimSpace(caller)
		}
	}

	// Add structured data for debug/trace levels
	if level <= DebugLevel && len(args) > 0 {
		processedArgs := make([]any, len(args))
		for i, arg := range args {
			processedArgs[i] = l.cloudFormatter.Format(arg)
		}
		logEntry["data"] = processedArgs
	}

	if data, err := json.Marshal(logEntry); err == nil {
		fmt.Fprintln(l.out, string(data))
	}
}

// logCloudWatch outputs CloudWatch-optimized logs
func (l *Logger) logCloudWatch(level Level, formatArgs bool, msg string, args ...any) {
	var processedArgs []any
	if formatArgs && level <= DebugLevel {
		processedArgs = make([]any, len(args))
		for i, arg := range args {
			processedArgs[i] = l.cloudFormatter.Format(arg)
		}
	} else {
		processedArgs = args
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	levelStr := level.String()

	var caller string
	if l.showCaller {
		caller = l.findCaller()
	}

	message := fmt.Sprintf(msg, processedArgs...)

	var fullMessage string
	if l.prefix != "" {
		fullMessage = fmt.Sprintf("[%s] %s [%s]%s: %s",
			timestamp, l.prefix, levelStr, caller, message)
	} else {
		fullMessage = fmt.Sprintf("[%s] [%s]%s: %s",
			timestamp, levelStr, caller, message)
	}

	fmt.Fprintln(l.out, fullMessage)
}

// logConsole outputs beautiful console logs
func (l *Logger) logConsole(level Level, formatArgs bool, msg string, args ...any) {
	var processedArgs []any
	if formatArgs && level <= DebugLevel {
		processedArgs = make([]any, len(args))
		for i, arg := range args {
			processedArgs[i] = l.debugFormatter.Format(arg)
		}
	} else {
		processedArgs = args
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := level.String()

	if l.colored {
		levelStr = level.Color() + levelStr + "\033[0m"
	}

	caller := l.findCaller()
	message := fmt.Sprintf(msg, processedArgs...)

	var fullMessage string
	if l.prefix != "" {
		fullMessage = fmt.Sprintf("[%s] %s [%s]%s: %s\n",
			timestamp, l.prefix, levelStr, caller, message)
	} else {
		fullMessage = fmt.Sprintf("[%s] [%s]%s: %s\n",
			timestamp, levelStr, caller, message)
	}

	fmt.Fprint(l.out, fullMessage)
}

// Trace logs a message at trace level
func (l *Logger) Trace(msg string, args ...any) {
	l.logInternal(TraceLevel, true, msg, args...)
}

// Debug logs a message at debug level
func (l *Logger) Debug(msg string, args ...any) {
	l.logInternal(DebugLevel, true, msg, args...)
}

// Info logs a message at info level
func (l *Logger) Info(msg string, args ...any) {
	l.logInternal(InfoLevel, false, msg, args...)
}

// Warn logs a message at warn level
func (l *Logger) Warn(msg string, args ...any) {
	l.logInternal(WarnLevel, false, msg, args...)
}

// Error logs a message at error level
func (l *Logger) Error(msg string, args ...any) {
	l.logInternal(ErrorLevel, false, msg, args...)
}

// Fatal logs a message at error level and exits
func (l *Logger) Fatal(msg string, args ...any) {
	l.logInternal(ErrorLevel, false, msg, args...)
	os.Exit(1)
}

// DebugStruct logs a struct with full debug formatting
func (l *Logger) DebugStruct(name string, value any) {
	if !l.IsLevelEnabled(DebugLevel) {
		return
	}

	switch l.format {
	case FormatJSON:
		logEntry := map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "DEBUG",
			"message":   fmt.Sprintf("%s = %s", name, l.cloudFormatter.Format(value)),
			"struct":    value,
		}
		if l.showCaller {
			caller := l.findCaller()
			if caller != "" {
				logEntry["caller"] = strings.TrimSpace(caller)
			}
		}
		if data, err := json.Marshal(logEntry); err == nil {
			fmt.Fprintln(l.out, string(data))
		}
	case FormatCloudWatch:
		formatted := l.cloudFormatter.Format(value)
		l.logCloudWatch(DebugLevel, false, "%s = %s", name, formatted)
	default:
		formatted := l.debugFormatter.Format(value)
		l.logConsole(DebugLevel, false, "%s = %s", name, formatted)
	}
}

// TraceStruct logs a struct with full debug formatting at trace level
func (l *Logger) TraceStruct(name string, value any) {
	if !l.IsLevelEnabled(TraceLevel) {
		return
	}

	switch l.format {
	case FormatJSON:
		logEntry := map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "TRACE",
			"message":   fmt.Sprintf("%s = %s", name, l.cloudFormatter.Format(value)),
			"struct":    value,
		}
		if l.showCaller {
			caller := l.findCaller()
			if caller != "" {
				logEntry["caller"] = strings.TrimSpace(caller)
			}
		}
		if data, err := json.Marshal(logEntry); err == nil {
			fmt.Fprintln(l.out, string(data))
		}
	case FormatCloudWatch:
		formatted := l.cloudFormatter.Format(value)
		l.logCloudWatch(TraceLevel, false, "%s = %s", name, formatted)
	default:
		formatted := l.debugFormatter.Format(value)
		l.logConsole(TraceLevel, false, "%s = %s", name, formatted)
	}
}
