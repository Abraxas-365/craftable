package logx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// CloudWatchFormatter formats logs for AWS CloudWatch
type CloudWatchFormatter struct {
	useJSON bool
}

// NewCloudWatchFormatter creates a CloudWatch-optimized formatter
func NewCloudWatchFormatter(useJSON bool) *CloudWatchFormatter {
	return &CloudWatchFormatter{
		useJSON: useJSON,
	}
}

// Format formats a value for CloudWatch (single line, no colors)
func (cf *CloudWatchFormatter) Format(v any) string {
	if cf.useJSON {
		return cf.formatJSON(v)
	}
	return cf.formatCompact(v)
}

// formatJSON converts to JSON for structured logging
func (cf *CloudWatchFormatter) formatJSON(v any) string {
	// Handle special types
	switch val := v.(type) {
	case error:
		if val != nil {
			return fmt.Sprintf(`{"error": "%s", "type": "%T"}`, val.Error(), val)
		}
		return "null"
	case time.Time:
		return fmt.Sprintf(`{"time": "%s"}`, val.Format(time.RFC3339))
	}

	// Try to marshal to JSON
	if data, err := json.Marshal(v); err == nil {
		return string(data)
	}

	// Fallback to string representation
	return fmt.Sprintf(`"%v"`, v)
}

// formatCompact creates a single-line compact representation
func (cf *CloudWatchFormatter) formatCompact(v any) string {
	return cf.formatValueCompact(reflect.ValueOf(v))
}

func (cf *CloudWatchFormatter) formatValueCompact(v reflect.Value) string {
	if !v.IsValid() {
		return "<nil>"
	}

	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "nil"
	}

	// Handle error interface
	if v.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if v.CanInterface() {
			if err, ok := v.Interface().(error); ok && err != nil {
				return fmt.Sprintf("Error(%q)", err.Error())
			}
		}
	}

	if v.Kind() == reflect.Ptr {
		return "&" + cf.formatValueCompact(v.Elem())
	}

	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Slice, reflect.Array:
		return cf.formatSliceCompact(v)
	case reflect.Map:
		return cf.formatMapCompact(v)
	case reflect.Struct:
		return cf.formatStructCompact(v)
	case reflect.Interface:
		if v.IsNil() {
			return "<nil>"
		}
		return cf.formatValueCompact(v.Elem())
	default:
		if v.CanInterface() {
			return fmt.Sprintf("%v", v.Interface())
		}
		return fmt.Sprintf("<%s>", v.Type().String())
	}
}

func (cf *CloudWatchFormatter) formatStructCompact(v reflect.Value) string {
	t := v.Type()

	// Handle special types
	switch t {
	case reflect.TypeOf(time.Time{}):
		if tm, ok := v.Interface().(time.Time); ok {
			return fmt.Sprintf("Time(%q)", tm.Format(time.RFC3339))
		}
	}

	var parts []string
	typeName := t.Name()
	if typeName == "" {
		typeName = "struct"
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		fieldStr := fmt.Sprintf("%s:%s", field.Name, cf.formatValueCompact(fieldValue))
		parts = append(parts, fieldStr)
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s{}", typeName)
	}

	return fmt.Sprintf("%s{%s}", typeName, strings.Join(parts, ","))
}

func (cf *CloudWatchFormatter) formatSliceCompact(v reflect.Value) string {
	length := v.Len()
	if length == 0 {
		return "[]"
	}

	// Handle byte slices
	if v.Type().Elem().Kind() == reflect.Uint8 {
		if data, ok := v.Interface().([]byte); ok {
			return fmt.Sprintf("[]byte(%q)", string(data))
		}
	}

	var parts []string
	for i := 0; i < length; i++ {
		parts = append(parts, cf.formatValueCompact(v.Index(i)))
	}

	return fmt.Sprintf("[%s]", strings.Join(parts, ","))
}

func (cf *CloudWatchFormatter) formatMapCompact(v reflect.Value) string {
	keys := v.MapKeys()
	if len(keys) == 0 {
		return "map{}"
	}

	var parts []string
	for _, key := range keys {
		keyStr := cf.formatValueCompact(key)
		valueStr := cf.formatValueCompact(v.MapIndex(key))
		parts = append(parts, fmt.Sprintf("%s:%s", keyStr, valueStr))
	}

	return fmt.Sprintf("map{%s}", strings.Join(parts, ","))
}

// DebugFormatter handles pretty printing of complex types for console output
type DebugFormatter struct {
	indent        int
	maxDepth      int
	showTypes     bool
	compactArrays bool
}

// NewDebugFormatter creates a new debug formatter
func NewDebugFormatter() *DebugFormatter {
	return &DebugFormatter{
		indent:        0,
		maxDepth:      10,
		showTypes:     true,
		compactArrays: true,
	}
}

// Format formats a value with debug information
func (df *DebugFormatter) Format(v any) string {
	return df.formatValue(reflect.ValueOf(v), 0)
}

// formatValue recursively formats a reflect.Value
func (df *DebugFormatter) formatValue(v reflect.Value, depth int) string {
	if depth > df.maxDepth {
		return "..."
	}

	if !v.IsValid() {
		return "<nil>"
	}

	// Handle nil pointers
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "nil"
	}

	// Handle error interface FIRST, before checking for structs
	if v.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if v.CanInterface() {
			if err, ok := v.Interface().(error); ok && err != nil {
				return fmt.Sprintf("Error(%q)", err.Error())
			}
		}
	}

	// Dereference pointers
	if v.Kind() == reflect.Ptr {
		return "&" + df.formatValue(v.Elem(), depth)
	}

	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", v.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())

	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())

	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())

	case reflect.Slice, reflect.Array:
		return df.formatSlice(v, depth)

	case reflect.Map:
		return df.formatMap(v, depth)

	case reflect.Struct:
		return df.formatStruct(v, depth)

	case reflect.Interface:
		if v.IsNil() {
			return "<nil>"
		}
		return df.formatValue(v.Elem(), depth)

	default:
		// For unknown types, try to get the interface and format it
		if v.CanInterface() {
			return fmt.Sprintf("%v", v.Interface())
		}
		return fmt.Sprintf("<%s>", v.Type().String())
	}
}

// formatStruct formats a struct with field names and values
func (df *DebugFormatter) formatStruct(v reflect.Value, depth int) string {
	t := v.Type()

	// Handle special types
	switch t {
	case reflect.TypeOf(time.Time{}):
		if tm, ok := v.Interface().(time.Time); ok {
			return fmt.Sprintf("Time(%q)", tm.Format(time.RFC3339))
		}
	}

	var parts []string
	typeName := t.Name()
	if typeName == "" {
		typeName = "struct"
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		fieldStr := fmt.Sprintf("%s: %s",
			field.Name,
			df.formatValue(fieldValue, depth+1))
		parts = append(parts, fieldStr)
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s{}", typeName)
	}

	// For small structs, use compact format
	if len(parts) <= 2 && depth > 0 {
		return fmt.Sprintf("%s{ %s }", typeName, strings.Join(parts, ", "))
	}

	// For larger structs or top-level, use multi-line format
	indent := strings.Repeat("  ", depth+1)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s{\n", typeName))
	for _, part := range parts {
		result.WriteString(fmt.Sprintf("%s%s,\n", indent, part))
	}
	result.WriteString(strings.Repeat("  ", depth) + "}")

	return result.String()
}

// formatSlice formats slices and arrays
func (df *DebugFormatter) formatSlice(v reflect.Value, depth int) string {
	length := v.Len()
	if length == 0 {
		return "[]"
	}

	// Handle byte slices specially
	if v.Type().Elem().Kind() == reflect.Uint8 {
		if data, ok := v.Interface().([]byte); ok {
			return fmt.Sprintf("[]byte(%q)", string(data))
		}
	}

	var parts []string
	for i := 0; i < length; i++ {
		parts = append(parts, df.formatValue(v.Index(i), depth+1))
	}

	// Compact format for simple types or short arrays
	if df.compactArrays && (length <= 5 || depth > 2) {
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	// Multi-line format for complex or long arrays
	indent := strings.Repeat("  ", depth+1)
	var result strings.Builder
	result.WriteString("[\n")
	for _, part := range parts {
		result.WriteString(fmt.Sprintf("%s%s,\n", indent, part))
	}
	result.WriteString(strings.Repeat("  ", depth) + "]")

	return result.String()
}

// formatMap formats maps
func (df *DebugFormatter) formatMap(v reflect.Value, depth int) string {
	keys := v.MapKeys()
	if len(keys) == 0 {
		return "map{}"
	}

	var parts []string
	for _, key := range keys {
		keyStr := df.formatValue(key, depth+1)
		valueStr := df.formatValue(v.MapIndex(key), depth+1)
		parts = append(parts, fmt.Sprintf("%s: %s", keyStr, valueStr))
	}

	// Compact format for small maps
	if len(parts) <= 3 && depth > 0 {
		return fmt.Sprintf("map{ %s }", strings.Join(parts, ", "))
	}

	// Multi-line format
	indent := strings.Repeat("  ", depth+1)
	var result strings.Builder
	result.WriteString("map{\n")
	for _, part := range parts {
		result.WriteString(fmt.Sprintf("%s%s,\n", indent, part))
	}
	result.WriteString(strings.Repeat("  ", depth) + "}")

	return result.String()
}
