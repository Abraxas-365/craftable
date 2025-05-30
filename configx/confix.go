package configx

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config represents the main configuration interface
type Config interface {
	// Get retrieves a configuration value by key
	Get(key string) Value

	// Set sets a configuration value
	Set(key string, val any)

	// Has checks if a configuration key exists
	Has(key string) bool

	// AllSettings returns all settings as a map
	AllSettings() map[string]any

	// AddSource adds a configuration source
	AddSource(source Source) Config

	// LoadAll reloads all configuration sources
	LoadAll() error

	// RequireEnv specifies environment variables that must be present
	RequireEnv(envVars ...string) error
}

// Source represents a configuration source
type Source interface {
	// Load loads configuration values from the source
	Load() (map[string]any, error)

	// Name returns the name of the source
	Name() string

	// Priority returns the priority of the source (higher values override lower)
	Priority() int
}

// Value wraps a configuration value and provides type conversion methods
type Value interface {
	// IsSet returns true if the value exists
	IsSet() bool

	// AsString returns the value as a string
	AsString() string

	// AsStringDefault returns the value as a string or a default value
	AsStringDefault(def string) string

	// AsInt returns the value as an int
	AsInt() int

	// AsIntDefault returns the value as an int or a default value
	AsIntDefault(def int) int

	// AsFloat returns the value as a float64
	AsFloat() float64

	// AsFloatDefault returns the value as a float64 or a default value
	AsFloatDefault(def float64) float64

	// AsBool returns the value as a bool
	AsBool() bool

	// AsBoolDefault returns the value as a bool or a default value
	AsBoolDefault(def bool) bool

	// AsDuration returns the value as a duration
	AsDuration() time.Duration

	// AsDurationDefault returns the value as a duration or a default value
	AsDurationDefault(def time.Duration) time.Duration

	// AsSlice returns the value as a slice of Values
	AsSlice() []Value

	// AsStringSlice returns the value as a string slice
	AsStringSlice() []string

	// AsIntSlice returns the value as an int slice
	AsIntSlice() []int

	// AsMap returns the value as a map
	AsMap() map[string]Value

	// AsStruct unmarshals the value into a struct
	AsStruct(target any) error
}

// Option is a function that configures a configuration
type Option func(*configuration)

// Builder provides a fluent API for building configuration
type Builder interface {
	// FromFile adds a file source
	FromFile(path string) Builder

	FromDotEnv(path string) Builder
	// FromEnv adds an environment variable source
	FromEnv(prefix string) Builder

	// FromMap adds a map source
	FromMap(values map[string]any, name string) Builder

	// WithDefaults adds default values
	WithDefaults(defaults map[string]any) Builder

	// WithAutoReload enables automatic reloading of configuration
	WithAutoReload(interval time.Duration) Builder

	// WithValidation adds validation to configuration
	WithValidation(validator func(config Config) error) Builder

	// WithOnChange registers a function to be called when a configuration value changes
	WithOnChange(hook func(key string, value any)) Builder

	// RequireEnv specifies environment variables that must be present
	RequireEnv(envVars ...string) Builder

	// Build builds the configuration
	Build() (Config, error)
}

//-----------------------------------------------------------------------------
// Implementation
//-----------------------------------------------------------------------------

// configuration is the concrete implementation of Config
type configuration struct {
	sync.RWMutex
	values        map[string]any
	sources       []Source
	isAutoReload  bool
	reloadSignal  chan struct{}
	reloadStop    chan struct{}
	reloadTimer   *time.Timer
	onChangeHooks []func(key string, value any)
	requiredEnvs  []string
}

// New creates a new Config instance
func New(opts ...Option) (Config, error) {
	cfg := &configuration{
		values:        make(map[string]any),
		sources:       make([]Source, 0),
		reloadSignal:  make(chan struct{}, 1),
		reloadStop:    make(chan struct{}),
		onChangeHooks: make([]func(key string, value any), 0),
		requiredEnvs:  make([]string, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Check for required environment variables
	if err := cfg.RequireEnv(cfg.requiredEnvs...); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Get retrieves a configuration value by key
func (c *configuration) Get(key string) Value {
	c.RLock()
	defer c.RUnlock()

	if key == "" {
		return newValue("", c.values)
	}

	return newValue(key, c.findValue(key))
}

// findValue searches for a key in the configuration, supporting nested keys with dot notation
func (c *configuration) findValue(key string) any {
	parts := strings.Split(key, ".")
	current := c.values

	for i, part := range parts {
		if v, ok := current[part]; ok {
			if i == len(parts)-1 {
				return v
			}

			if m, ok := v.(map[string]any); ok {
				current = m
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return nil
}

// Set sets a configuration value
func (c *configuration) Set(key string, val any) {
	c.Lock()
	defer c.Unlock()

	parts := strings.Split(key, ".")
	current := c.values

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = val

			// Notify change hooks
			for _, hook := range c.onChangeHooks {
				go hook(key, val)
			}

			return
		}

		if _, ok := current[part]; !ok {
			current[part] = make(map[string]any)
		}

		if m, ok := current[part].(map[string]any); ok {
			current = m
		} else {
			// Cannot navigate further, overwrite with a new map
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}
}

// Has checks if a configuration key exists
func (c *configuration) Has(key string) bool {
	c.RLock()
	defer c.RUnlock()

	return c.findValue(key) != nil
}

// AllSettings returns all settings as a map
func (c *configuration) AllSettings() map[string]any {
	c.RLock()
	defer c.RUnlock()

	// Return a copy of the settings to prevent modification
	return deepCopyMap(c.values)
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))

	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = deepCopyMap(val)
		case []any:
			result[k] = deepCopySlice(val)
		default:
			result[k] = val
		}
	}

	return result
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))

	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = deepCopyMap(val)
		case []any:
			result[i] = deepCopySlice(val)
		default:
			result[i] = val
		}
	}

	return result
}

// AddSource adds a configuration source
func (c *configuration) AddSource(source Source) Config {
	c.Lock()
	defer c.Unlock()

	c.sources = append(c.sources, source)

	// Sort sources by priority
	sortSourcesByPriority(c.sources)

	// Load the new source immediately
	if data, err := source.Load(); err == nil {
		c.mergeMap(data)
	}

	return c
}

// sortSourcesByPriority sorts sources by priority (higher priority last)
func sortSourcesByPriority(sources []Source) {
	// Simple bubble sort for clarity
	for i := 0; i < len(sources); i++ {
		for j := i + 1; j < len(sources); j++ {
			if sources[i].Priority() > sources[j].Priority() {
				sources[i], sources[j] = sources[j], sources[i]
			}
		}
	}
}

// mergeMap merges a map of values into the configuration
func (c *configuration) mergeMap(data map[string]any) {
	for k, v := range data {
		if m, ok := v.(map[string]any); ok {
			// Handle nested maps
			if existing, ok := c.values[k]; ok {
				if existingMap, ok := existing.(map[string]any); ok {
					// If both are maps, merge them
					merged := deepCopyMap(existingMap)
					mergeMapRecursive(merged, m)
					c.values[k] = merged
					continue
				}
			}
		}

		// For non-map values or if the existing value is not a map, just replace
		c.values[k] = v
	}
}

// mergeMapRecursive recursively merges maps
func mergeMapRecursive(dst, src map[string]any) {
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			// If it's a map, merge recursively
			if dstVal, ok := dst[k]; ok {
				if dstMap, ok := dstVal.(map[string]any); ok {
					// Both are maps, merge them
					mergeMapRecursive(dstMap, srcMap)
					continue
				}
			}

			// If the destination is not a map or doesn't exist, create a new map
			dst[k] = deepCopyMap(srcMap)
		} else {
			// For non-map values, just replace
			dst[k] = v
		}
	}
}

// LoadAll reloads all configuration sources
func (c *configuration) LoadAll() error {
	c.Lock()
	defer c.Unlock()

	// Create a new values map
	newValues := make(map[string]any)

	// Load each source in order of priority
	for _, source := range c.sources {
		data, err := source.Load()
		if err != nil {
			return fmt.Errorf("error loading from source %s: %w", source.Name(), err)
		}

		// Merge into new values
		for k, v := range data {
			if m, ok := v.(map[string]any); ok {
				// Handle nested maps
				if existing, ok := newValues[k]; ok {
					if existingMap, ok := existing.(map[string]any); ok {
						// If both are maps, merge them
						merged := deepCopyMap(existingMap)
						mergeMapRecursive(merged, m)
						newValues[k] = merged
						continue
					}
				}
			}

			// For non-map values or if the existing value is not a map, just replace
			newValues[k] = v
		}
	}

	// Find changed values for hooks
	changedKeys := make(map[string]any)
	collectChangedKeys("", c.values, newValues, changedKeys)

	// Update values
	c.values = newValues

	// Trigger hooks for changed values
	for k, v := range changedKeys {
		for _, hook := range c.onChangeHooks {
			go hook(k, v)
		}
	}

	return nil
}

// collectChangedKeys recursively collects keys that have changed between two maps
func collectChangedKeys(prefix string, oldMap, newMap map[string]any, changedKeys map[string]any) {
	// Check keys in newMap
	for k, newVal := range newMap {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if oldVal, exists := oldMap[k]; !exists {
			// Key doesn't exist in old map
			changedKeys[key] = newVal
		} else if !reflect.DeepEqual(oldVal, newVal) {
			if oldMap, ok := oldVal.(map[string]any); ok {
				if newMap, ok := newVal.(map[string]any); ok {
					// Both are maps, check recursively
					collectChangedKeys(key, oldMap, newMap, changedKeys)
					continue
				}
			}

			// Values are different
			changedKeys[key] = newVal
		}
	}

	// Check for keys in oldMap that don't exist in newMap
	for k := range oldMap {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if _, exists := newMap[k]; !exists {
			changedKeys[key] = nil
		}
	}
}

// RequireEnv specifies environment variables that must be present
func (c *configuration) RequireEnv(envVars ...string) error {
	// Create a set of required environment variables to avoid duplicates
	requiredSet := make(map[string]bool)
	for _, env := range c.requiredEnvs {
		requiredSet[env] = true
	}

	// Add new environment variables if they don't already exist
	for _, env := range envVars {
		if !requiredSet[env] {
			c.requiredEnvs = append(c.requiredEnvs, env)
			requiredSet[env] = true
		}
	}

	// Check if all required environment variables are present
	var missing []string
	for env := range requiredSet {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

//-----------------------------------------------------------------------------
// Value implementation
//-----------------------------------------------------------------------------

// value implements the Value interface
type value struct {
	key string
	val any
}

// newValue creates a new Value instance
func newValue(key string, val any) Value {
	return &value{
		key: key,
		val: val,
	}
}

// IsSet returns true if the value exists
func (v *value) IsSet() bool {
	return v.val != nil
}

// AsString returns the value as a string
func (v *value) AsString() string {
	return v.AsStringDefault("")
}

// AsStringDefault returns the value as a string or a default value
func (v *value) AsStringDefault(def string) string {
	if !v.IsSet() {
		return def
	}

	switch val := v.val.(type) {
	case string:
		return val
	case int, int64, uint, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", val)
	default:
		return def
	}
}

// AsInt returns the value as an int
func (v *value) AsInt() int {
	return v.AsIntDefault(0)
}

// AsIntDefault returns the value as an int or a default value
func (v *value) AsIntDefault(def int) int {
	if !v.IsSet() {
		return def
	}

	switch val := v.val.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}

	return def
}

// AsFloat returns the value as a float64
func (v *value) AsFloat() float64 {
	return v.AsFloatDefault(0)
}

// AsFloatDefault returns the value as a float64 or a default value
func (v *value) AsFloatDefault(def float64) float64 {
	if !v.IsSet() {
		return def
	}

	switch val := v.val.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}

	return def
}

// AsBool returns the value as a bool
func (v *value) AsBool() bool {
	return v.AsBoolDefault(false)
}

// AsBoolDefault returns the value as a bool or a default value
func (v *value) AsBoolDefault(def bool) bool {
	if !v.IsSet() {
		return def
	}

	switch val := v.val.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case string:
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}

		// Also accept "yes", "y", "1" as true
		switch val {
		case "yes", "y", "Y", "1":
			return true
		}
	}

	return def
}

// AsDuration returns the value as a duration
func (v *value) AsDuration() time.Duration {
	return v.AsDurationDefault(0)
}

// AsDurationDefault returns the value as a duration or a default value
func (v *value) AsDurationDefault(def time.Duration) time.Duration {
	if !v.IsSet() {
		return def
	}

	switch val := v.val.(type) {
	case time.Duration:
		return val
	case int, int64, float64:
		// Assume milliseconds
		return time.Duration(v.AsInt()) * time.Millisecond
	case string:
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}

	return def
}

// AsSlice returns the value as a slice of Values
func (v *value) AsSlice() []Value {
	if !v.IsSet() {
		return []Value{}
	}

	switch val := v.val.(type) {
	case []any:
		result := make([]Value, len(val))
		for i, item := range val {
			result[i] = newValue(fmt.Sprintf("%s[%d]", v.key, i), item)
		}
		return result
	}

	return []Value{v}
}

// AsStringSlice returns the value as a string slice
func (v *value) AsStringSlice() []string {
	values := v.AsSlice()
	result := make([]string, len(values))

	for i, val := range values {
		result[i] = val.AsString()
	}

	return result
}

// AsIntSlice returns the value as an int slice
func (v *value) AsIntSlice() []int {
	values := v.AsSlice()
	result := make([]int, len(values))

	for i, val := range values {
		result[i] = val.AsInt()
	}

	return result
}

// AsMap returns the value as a map
func (v *value) AsMap() map[string]Value {
	if !v.IsSet() {
		return map[string]Value{}
	}

	switch val := v.val.(type) {
	case map[string]any:
		result := make(map[string]Value, len(val))
		for k, item := range val {
			// Use v.key (from the outer struct) and k (from the loop)
			result[k] = newValue(fmt.Sprintf("%s.%s", v.key, k), item)
		}
		return result
	}

	return map[string]Value{}
}

// AsStruct unmarshals the value into a struct
func (v *value) AsStruct(target any) error {
	if !v.IsSet() {
		return fmt.Errorf("value not set")
	}

	// Convert to JSON and then unmarshal
	jsonData, err := json.Marshal(v.val)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal configuration to struct: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Builder implementation
// -----------------------------------------------------------------------------
const (
	PriorityDefault = 10 // Lowest priority
	PriorityEnv     = 20
	PriorityFile    = 30
	PriorityDotEnv  = 25
	PriorityMap     = 40 // Highest priority
)

// builder implements the Builder interface
type builder struct {
	options     []Option
	requiredEnv []string
}

// NewBuilder creates a new configuration builder
func NewBuilder() Builder {
	return &builder{
		options:     make([]Option, 0),
		requiredEnv: make([]string, 0),
	}
}

// FromFile adds a file source
func (b *builder) FromFile(path string) Builder {
	b.options = append(b.options, WithFile(path, len(b.options)+1))
	return b
}

// FromEnv adds an environment variable source
func (b *builder) FromEnv(prefix string) Builder {
	b.options = append(b.options, WithEnv(prefix, PriorityEnv))
	return b
}

func (b *builder) FromDotEnv(path string) Builder {
	b.options = append(b.options, WithDotEnv(path, PriorityDotEnv))
	return b
}

// FromMap adds a map source
func (b *builder) FromMap(values map[string]any, name string) Builder {
	b.options = append(b.options, WithMap(values, name, PriorityMap))
	return b
}

// WithDefaults adds default values
func (b *builder) WithDefaults(defaults map[string]any) Builder {
	b.options = append(b.options, WithMap(defaults, "defaults", PriorityDefault))
	return b
}

// WithAutoReload enables automatic reloading of configuration
func (b *builder) WithAutoReload(interval time.Duration) Builder {
	b.options = append(b.options, WithAutoReload(interval))
	return b
}

// WithValidation adds validation to configuration
func (b *builder) WithValidation(validator func(config Config) error) Builder {
	b.options = append(b.options, WithValidation(validator))
	return b
}

// WithOnChange registers a function to be called when a configuration value changes
func (b *builder) WithOnChange(hook func(key string, value any)) Builder {
	b.options = append(b.options, WithOnChange(hook))
	return b
}

// WithDotEnv adds a .env file source
func WithDotEnv(path string, priority int) Option {
	return func(c *configuration) {
		c.AddSource(NewDotEnvSource(path, priority))
	}
}

// RequireEnv specifies environment variables that must be present
func (b *builder) RequireEnv(envVars ...string) Builder {
	b.requiredEnv = append(b.requiredEnv, envVars...)
	return b
}

// Build builds the configuration
func (b *builder) Build() (Config, error) {
	// First apply all the configuration options
	cfg, err := New(b.options...)
	if err != nil {
		return nil, err
	}

	// Then check required environment variables
	if len(b.requiredEnv) > 0 {
		if err := cfg.RequireEnv(b.requiredEnv...); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

//-----------------------------------------------------------------------------
// Options
//-----------------------------------------------------------------------------

// WithSource adds a configuration source
func WithSource(source Source) Option {
	return func(c *configuration) {
		c.AddSource(source)
	}
}

// WithFile adds a file source
func WithFile(path string, priority int) Option {
	return func(c *configuration) {
		// This would normally call NewFileSource, which would be in sources.go
		//TODO
		log.Println("File source not implemented")
	}
}

// WithEnv adds an environment variable source
func WithEnv(prefix string, priority int) Option {
	return func(c *configuration) {
		c.AddSource(NewEnvSource(prefix, priority))
	}
}

// WithMap adds a map source
func WithMap(values map[string]any, name string, priority int) Option {
	return func(c *configuration) {
		// This would normally call NewMapSource, which would be in sources.go
		//TODO
		log.Println("Map source not implemented")
	}
}

// WithDefaults adds default values
func WithDefaults(defaults map[string]any) Option {
	return WithMap(defaults, "defaults", 0)
}

// WithAutoReload enables automatic reloading of configuration
func WithAutoReload(interval time.Duration) Option {
	return func(c *configuration) {
		c.isAutoReload = true

		// Stop previous timer if it exists
		if c.reloadTimer != nil {
			c.reloadTimer.Stop()
		}

		// Create new timer
		c.reloadTimer = time.NewTimer(interval)

		// Start goroutine for auto-reload
		go func() {
			for {
				select {
				case <-c.reloadTimer.C:
					c.LoadAll()
					c.reloadTimer.Reset(interval)
				case <-c.reloadSignal:
					c.LoadAll()
					c.reloadTimer.Reset(interval)
				case <-c.reloadStop:
					c.reloadTimer.Stop()
					return
				}
			}
		}()
	}
}

// WithOnChange registers a function to be called when a configuration value changes
func WithOnChange(hook func(key string, value any)) Option {
	return func(c *configuration) {
		c.onChangeHooks = append(c.onChangeHooks, hook)
	}
}

// WithValidation adds validation to configuration
func WithValidation(validator func(config Config) error) Option {
	return func(c *configuration) {
		// Add a hook to validate after loading
		c.onChangeHooks = append(c.onChangeHooks, func(key string, value any) {
			if err := validator(c); err != nil {
				// Log validation error
				fmt.Fprintf(os.Stderr, "Configuration validation error: %s\n", err)
			}
		})
	}
}

func WithRequiredEnvs(envVars []string) Option {
	return func(c *configuration) {
		// This will handle duplicates and check for missing variables
		c.RequireEnv(envVars...)
	}
}
