package configx

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment variables source
// ===========================

// EnvSource loads configuration from environment variables
type EnvSource struct {
	prefix   string
	priority int
}

// NewEnvSource creates a new environment variable source
func NewEnvSource(prefix string, priority int) Source {
	return &EnvSource{
		prefix:   prefix,
		priority: priority,
	}
}

// Load loads configuration values from environment variables
func (s *EnvSource) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		// If prefix is specified, only include variables with the prefix
		if s.prefix != "" && !strings.HasPrefix(key, s.prefix) {
			continue
		}

		// Remove prefix from key if it exists
		if s.prefix != "" {
			key = strings.TrimPrefix(key, s.prefix)
		}

		// Convert key to lowercase and replace underscores with dots for nesting
		key = strings.ToLower(key)

		// Here's the crucial change:
		// We need to handle nested structures differently
		parts = strings.Split(key, "_")
		if len(parts) > 1 {
			// This is a nested key like SERVER_PORT
			typedValue := s.convertValue(value)

			// Create nested maps as needed
			currentMap := result
			for i := 0; i < len(parts)-1; i++ {
				partKey := parts[i]
				if _, exists := currentMap[partKey]; !exists {
					currentMap[partKey] = make(map[string]interface{})
				}
				if nestedMap, ok := currentMap[partKey].(map[string]interface{}); ok {
					currentMap = nestedMap
				} else {
					// If it exists but isn't a map, overwrite with a map
					newMap := make(map[string]interface{})
					currentMap[partKey] = newMap
					currentMap = newMap
				}
			}

			// Set the value in the innermost map
			lastPart := parts[len(parts)-1]
			currentMap[lastPart] = typedValue
		} else {
			// This is a simple key like DEBUG
			result[key] = s.convertValue(value)
		}
	}

	return result, nil
}

// setNestedValue sets a value in a nested map structure based on a dot-separated key
func (s *EnvSource) setNestedValue(result map[string]interface{}, key, value string) {
	// Try to convert the value to a more specific type
	typedValue := s.convertValue(value)

	// Set the value in the map
	result[key] = typedValue
}

// convertValue attempts to convert a string value to a more appropriate type
// convertValue attempts to convert a string value to a more appropriate type
func (s *EnvSource) convertValue(value string) interface{} {
	// Try boolean
	if value == "true" || value == "TRUE" || value == "yes" || value == "YES" || value == "1" {
		return true
	}
	if value == "false" || value == "FALSE" || value == "no" || value == "NO" || value == "0" {
		return false
	}

	// Try integer
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Default to string
	return value
}

// Name returns the name of the source
func (s *EnvSource) Name() string {
	return fmt.Sprintf("env(%s)", s.prefix)
}

// Priority returns the priority of the source
func (s *EnvSource) Priority() int {
	return s.priority
}

// DotEnv file source
// ===========================

// DotEnvSource loads configuration from a .env file
type DotEnvSource struct {
	path     string
	priority int
}

// NewDotEnvSource creates a new .env file source
func NewDotEnvSource(path string, priority int) Source {
	return &DotEnvSource{
		path:     path,
		priority: priority,
	}
}

// Load loads configuration values from a .env file
func (s *DotEnvSource) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) > 1 && (value[0] == '"' && value[len(value)-1] == '"' ||
			value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}

		// Convert the key to lowercase and replace underscores with dots
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", ".")

		// Handle nested keys
		parseEnvValue(key, value, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return result, nil
}

// Name returns the name of the source
func (s *DotEnvSource) Name() string {
	return fmt.Sprintf("dotenv(%s)", s.path)
}

// Priority returns the priority of the source
func (s *DotEnvSource) Priority() int {
	return s.priority
}

// Helper functions
// ===========================

// parseEnvValue parses an environment variable value and adds it to the result map
func parseEnvValue(key, value string, result map[string]interface{}) {
	// Try to parse the value as a boolean
	if value == "true" || value == "false" {
		result[key] = value == "true"
		return
	}

	// Try to parse as an integer
	if intVal, err := strconv.Atoi(value); err == nil {
		result[key] = intVal
		return
	}

	// Try to parse as a float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		result[key] = floatVal
		return
	}

	// Default to string
	result[key] = value
}

// Add these two options to the main configx.go file

// Map Source implementation
// ===========================

// MapSource loads configuration from a map
type MapSource struct {
	values   map[string]interface{}
	name     string
	priority int
}

// NewMapSource creates a new map source
func NewMapSource(values map[string]interface{}, name string, priority int) Source {
	// Create a deep copy of the map to prevent modification
	copiedValues := deepCopyMap(values)
	return &MapSource{
		values:   copiedValues,
		name:     name,
		priority: priority,
	}
}

// Load loads configuration values from the map
func (s *MapSource) Load() (map[string]interface{}, error) {
	return deepCopyMap(s.values), nil
}

// Name returns the name of the source
func (s *MapSource) Name() string {
	return s.name
}

// Priority returns the priority of the source
func (s *MapSource) Priority() int {
	return s.priority
}

// Additional option implementations
// ===========================

// withAutoReload enables automatic configuration reloading
func withAutoReload(interval time.Duration) Option {
	return func(c *configuration) {
		c.isAutoReload = true

		// Start a timer to reload configuration periodically
		c.reloadTimer = time.NewTimer(interval)

		go func() {
			for {
				select {
				case <-c.reloadTimer.C:
					if err := c.LoadAll(); err != nil {
						fmt.Printf("Error reloading configuration: %v\n", err)
					}
					c.reloadTimer.Reset(interval)
				case <-c.reloadStop:
					c.reloadTimer.Stop()
					return
				}
			}
		}()
	}
}

// withValidator adds validation to the configuration
func withValidator(validator func(config Config) error) Option {
	return func(c *configuration) {
		// Apply validation immediately
		if err := validator(c); err != nil {
			fmt.Printf("Configuration validation failed: %v\n", err)
		}

		// Store validator for future validation
		origHooks := c.onChangeHooks
		c.onChangeHooks = append(origHooks, func(key string, value interface{}) {
			if err := validator(c); err != nil {
				fmt.Printf("Configuration validation failed after change: %v\n", err)
			}
		})
	}
}

// withOnChange registers a hook for configuration changes
func withOnChange(hook func(key string, value interface{})) Option {
	return func(c *configuration) {
		c.onChangeHooks = append(c.onChangeHooks, hook)
	}
}
