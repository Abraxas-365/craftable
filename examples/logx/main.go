package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/logx"
)

// Example structs to demonstrate formatting
type User struct {
	ID        int                    `json:"id"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	CreatedAt time.Time              `json:"created_at"`
	Profile   *Profile               `json:"profile,omitempty"`
	Tags      []string               `json:"tags"`
	Settings  map[string]interface{} `json:"settings"`
	IsActive  bool                   `json:"is_active"`
}

type Profile struct {
	Bio     string         `json:"bio"`
	Age     int            `json:"age"`
	Website string         `json:"website,omitempty"`
	Scores  map[string]int `json:"scores"`
}

// Custom error type
type DatabaseError struct {
	Code      int
	Message   string
	Query     string
	Timestamp time.Time
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database error [%d]: %s", e.Code, e.Message)
}

type ValidationError struct {
	Field  string
	Value  interface{}
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (value: %v)", e.Field, e.Reason, e.Value)
}

func main() {
	// Environment Variables - you can set these before running:
	//
	// 🎨 CONSOLE FORMAT (Default - Beautiful for Local Development):
	// LOG_LEVEL=DEBUG go run main.go
	// LOG_LEVEL=TRACE LOG_COLOR=true LOG_CALLER=true go run main.go
	//
	// ☁️ CLOUDWATCH FORMAT (Single-line, no colors, optimized for AWS):
	// LOG_FORMAT=cloudwatch LOG_LEVEL=INFO go run main.go
	// LOG_FORMAT=cloudwatch LOG_LEVEL=DEBUG LOG_CALLER=false go run main.go
	//
	// 📊 JSON FORMAT (Structured logging for aggregation systems):
	// LOG_FORMAT=json LOG_LEVEL=DEBUG go run main.go
	// LOG_FORMAT=json LOG_LEVEL=INFO LOG_CALLER=true go run main.go
	//
	// 🔇 MINIMAL LOGGING:
	// LOG_LEVEL=ERROR LOG_COLOR=false go run main.go
	// LOG_LEVEL=OFF go run main.go
	//
	// 🐛 MAXIMUM VERBOSITY:
	// LOG_LEVEL=TRACE LOG_FORMAT=console go run main.go

	logx.Info("🚀 Application starting...")
	logx.Info("Current log level: %s", getCurrentLogLevel())

	// Demonstrate different log levels
	demonstrateLogLevels()

	// Create sample data
	user := createSampleUser()
	config := createSampleConfig()

	// Demonstrate struct logging
	demonstrateStructLogging(user, config)

	// Demonstrate error logging
	demonstrateErrorLogging()

	// Demonstrate collections (slices, maps, arrays)
	demonstrateCollections()

	// Demonstrate nil and pointer handling
	demonstrateNilAndPointers()

	// Demonstrate nested structures
	demonstrateNestedStructures()

	// Performance test (optional)
	if logx.IsLevelEnabled(logx.TraceLevel) {
		demonstratePerformance()
	}

	logx.Info("✅ Application finished successfully")
}

func getCurrentLogLevel() string {
	levels := []logx.Level{
		logx.TraceLevel,
		logx.DebugLevel,
		logx.InfoLevel,
		logx.WarnLevel,
		logx.ErrorLevel,
	}

	for _, level := range levels {
		if logx.IsLevelEnabled(level) {
			return level.String()
		}
	}
	return "OFF"
}

func demonstrateLogLevels() {
	logx.Trace("🔍 This is a TRACE message - very detailed debugging info")
	logx.Debug("🐛 This is a DEBUG message - general debugging info")
	logx.Info("ℹ️  This is an INFO message - general information")
	logx.Warn("⚠️  This is a WARN message - something might be wrong")
	logx.Error("❌ This is an ERROR message - something is definitely wrong")

	// Note: Fatal would exit the program, so it's commented out
	// logx.Fatal("💀 This is a FATAL message - application will exit")
}

func demonstrateStructLogging(user User, config map[string]interface{}) {
	logx.Info("--- Struct Logging Examples ---")

	// Automatic struct formatting in debug messages
	logx.Debug("👤 Processing user: %v", user)
	logx.Trace("👤 User details (trace level): %v", user)

	// Explicit struct logging (cleaner API)
	logx.DebugStruct("user", user)
	logx.DebugStruct("config", config)

	// Mixed logging with context
	logx.Debug("User %s (ID: %d) has profile: %v", user.Name, user.ID, user.Profile)
}

func demonstrateErrorLogging() {
	logx.Info("--- Error Logging Examples ---")

	// Standard errors
	err1 := errors.New("simple error message")
	logx.Debug("Standard error: %v", err1)
	logx.Error("Failed to process request: %v", err1)

	// Wrapped errors
	err2 := fmt.Errorf("failed to save user: %w", err1)
	logx.Error("Wrapped error: %v", err2)

	// Custom error with value receiver
	err3 := DatabaseError{
		Code:      500,
		Message:   "connection timeout",
		Query:     "SELECT * FROM users WHERE id = ?",
		Timestamp: time.Now(),
	}
	logx.Error("Database operation failed: %v", err3)
	logx.DebugStruct("db_error", err3)

	// Custom error with pointer receiver
	err4 := &ValidationError{
		Field:  "email",
		Value:  "not-an-email",
		Reason: "invalid email format",
	}
	logx.Error("Validation failed: %v", err4)
	logx.DebugStruct("validation_error", err4)

	// Error in context
	userID := 123
	logx.Error("Failed to authenticate user %d: %v", userID, err1)
}

func demonstrateCollections() {
	logx.Info("--- Collections Logging Examples ---")

	// Simple slices
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	logx.Debug("Numbers: %v", numbers)

	// String slices
	tags := []string{"golang", "rust", "python", "javascript", "typescript"}
	logx.Debug("Programming languages: %v", tags)

	// Byte slices (special handling)
	data := []byte("Hello, World! This is binary data.")
	logx.Debug("Binary data: %v", data)

	// Maps with mixed types
	metadata := map[string]interface{}{
		"version":     "1.0.0",
		"debug":       true,
		"max_retries": 3,
		"timeout":     30.5,
		"features":    []string{"auth", "logging", "metrics"},
		"database": map[string]string{
			"driver": "postgres",
			"host":   "localhost",
			"port":   "5432",
		},
	}
	logx.DebugStruct("metadata", metadata)

	// Empty collections
	emptySlice := []string{}
	emptyMap := map[string]int{}
	logx.Debug("Empty slice: %v", emptySlice)
	logx.Debug("Empty map: %v", emptyMap)
}

func demonstrateNilAndPointers() {
	logx.Info("--- Nil and Pointer Handling Examples ---")

	// Nil pointer
	var nilUser *User
	logx.Debug("Nil user pointer: %v", nilUser)

	// Valid pointer
	user := &User{
		ID:   456,
		Name: "Jane Doe",
		Tags: []string{"admin"},
	}
	logx.Debug("Valid user pointer: %v", user)

	// Nil interface
	var nilError error
	logx.Debug("Nil error: %v", nilError)

	// User with nil profile
	userWithNilProfile := User{
		ID:      789,
		Name:    "Bob Smith",
		Profile: nil, // nil pointer in struct
		Tags:    []string{},
		Settings: map[string]interface{}{
			"theme": "light",
		},
	}
	logx.DebugStruct("user_with_nil_profile", userWithNilProfile)
}

func demonstrateNestedStructures() {
	logx.Info("--- Nested Structures Examples ---")

	// Deeply nested structure
	type Company struct {
		Name      string
		Founded   time.Time
		Employees []User
		Metadata  map[string]interface{}
	}

	company := Company{
		Name:    "TechCorp Inc.",
		Founded: time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		Employees: []User{
			createSampleUser(),
			{
				ID:       999,
				Name:     "Admin User",
				Email:    "admin@techcorp.com",
				IsActive: true,
				Profile: &Profile{
					Bio: "System Administrator",
					Age: 35,
					Scores: map[string]int{
						"sysadmin": 100,
						"security": 95,
					},
				},
			},
		},
		Metadata: map[string]interface{}{
			"industry": "Technology",
			"size":     "startup",
			"remote":   true,
		},
	}

	logx.DebugStruct("company", company)
	logx.Trace("Company trace logging: %v", company)
}

func demonstratePerformance() {
	logx.Info("--- Performance Test (Trace Level Only) ---")

	start := time.Now()

	// Log many messages to test performance
	for i := 0; i < 100; i++ {
		user := User{
			ID:       i,
			Name:     fmt.Sprintf("User %d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			IsActive: i%2 == 0,
			Tags:     []string{"test", "performance"},
		}

		if i%10 == 0 {
			logx.Trace("Processing user %d: %v", i, user)
		}
	}

	duration := time.Since(start)
	logx.Info("Performance test completed in %v", duration)
}

func createSampleUser() User {
	return User{
		ID:        123,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Tags:      []string{"golang", "rust", "python", "developer"},
		Settings: map[string]interface{}{
			"theme":         "dark",
			"notifications": true,
			"timeout":       30.5,
			"language":      "en",
		},
		Profile: &Profile{
			Bio:     "Software developer who loves clean code and good documentation",
			Age:     30,
			Website: "https://johndoe.dev",
			Scores: map[string]int{
				"golang":     95,
				"rust":       85,
				"python":     90,
				"javascript": 80,
				"docker":     88,
			},
		},
	}
}

func createSampleConfig() map[string]interface{} {
	return map[string]interface{}{
		"server": map[string]interface{}{
			"host":         "localhost",
			"port":         8080,
			"ssl_enabled":  false,
			"read_timeout": "30s",
		},
		"database": map[string]interface{}{
			"driver":          "postgres",
			"host":            "db.example.com",
			"port":            5432,
			"database":        "myapp",
			"ssl_mode":        "require",
			"max_connections": 100,
		},
		"features": []string{
			"authentication",
			"authorization",
			"logging",
			"metrics",
			"rate_limiting",
		},
		"debug": map[string]interface{}{
			"enabled":     true,
			"log_queries": true,
			"profiling":   false,
		},
	}
}

