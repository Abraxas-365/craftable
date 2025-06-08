// Package logx provides structured logging with environment variable configuration,
// beautiful debug formatting for Go structs (similar to Rust's debug formatting),
// and AWS CloudWatch optimized output.
//
// Environment Variables:
//   - LOG_LEVEL: Set the minimum log level (TRACE, DEBUG, INFO, WARN, ERROR, OFF)
//   - LOG_FORMAT: Set output format (console, cloudwatch, json)
//   - LOG_COLOR: Enable/disable colored output (true/false, default: true)
//   - LOG_CALLER: Enable/disable caller information (true/false, default: true)
//
// Basic Usage:
//
//	logx.Info("Server starting on port %d", 8080)
//	logx.Error("Failed to connect to database: %v", err)
//
// Debug Formatting:
//
//	// Automatic struct formatting at DEBUG/TRACE levels
//	logx.Debug("User data: %v", user)
//
//	// Explicit struct logging
//	logx.DebugStruct("user", user)
//	logx.TraceStruct("config", config)
//
// Format Examples:
//
//	Console Format (default - beautiful for local development):
//	LOG_LEVEL=DEBUG go run main.go
//	[2025-06-08 18:57:52] [DEBUG] main.go:64: User data: User{
//	  ID: 123,
//	  Name: "John Doe",
//	  Email: "john@example.com",
//	}
//
//	CloudWatch Format (single-line, no colors, optimized for AWS):
//	LOG_FORMAT=cloudwatch LOG_LEVEL=DEBUG go run main.go
//	[2025-06-08T18:57:52.000Z] [DEBUG] main.go:64: User data: User{ID:123,Name:"John Doe",Email:"john@example.com"}
//
//	JSON Format (structured logging for log aggregation):
//	LOG_FORMAT=json LOG_LEVEL=DEBUG go run main.go
//	{"timestamp":"2025-06-08T18:57:52Z","level":"DEBUG","message":"User data","caller":"main.go:64","data":[{...}]}
//
// Features:
//   - Environment variable configuration
//   - Beautiful struct formatting (similar to Rust's {:?})
//   - Multiple output formats (console, cloudwatch, json)
//   - Colored output with customizable colors
//   - Caller information (file:line)
//   - Multiple log levels (TRACE, DEBUG, INFO, WARN, ERROR)
//   - Support for nested structs, maps, slices, and pointers
//   - Special formatting for errors and time.Time
//   - Both global and instance-based loggers
//   - AWS CloudWatch optimized output
//   - Structured JSON logging for log aggregation systems
package logx
