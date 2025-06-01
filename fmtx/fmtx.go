package fmtx

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unsafe"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	Gray      = "\033[90m"
	BrightRed = "\033[91m"
)

// DebugOptions provides comprehensive control over debug output
type DebugOptions struct {
	MaxDepth         int                                         // Maximum nesting depth (0 = unlimited)
	ShowPrivate      bool                                        // Show unexported fields
	CompactMode      bool                                        // More compact output
	ShowTypes        bool                                        // Show type information
	ShowAddresses    bool                                        // Show memory addresses for pointers
	ShowSizes        bool                                        // Show sizes for slices/maps/strings
	UseColors        bool                                        // Use ANSI colors for output
	MaxStringLength  int                                         // Truncate strings longer than this (0 = no limit)
	MaxSliceLength   int                                         // Truncate slices longer than this (0 = no limit)
	SortMapKeys      bool                                        // Sort map keys for consistent output
	CustomFormatters map[reflect.Type]func(reflect.Value) string // Custom formatters for specific types
	FieldFilter      func(reflect.StructField) bool              // Filter which fields to show
	Indent           string                                      // Custom indentation string (default: "    ")
}

// DefaultOptions returns sensible default options
func DefaultOptions() DebugOptions {
	return DebugOptions{
		MaxDepth:        10,
		ShowPrivate:     false,
		CompactMode:     false,
		ShowTypes:       false,
		ShowAddresses:   false,
		ShowSizes:       false,
		UseColors:       false,
		MaxStringLength: 100,
		MaxSliceLength:  10,
		SortMapKeys:     true,
		Indent:          "    ",
	}
}

// PrettyOptions returns options optimized for pretty printing
func PrettyOptions() DebugOptions {
	opts := DefaultOptions()
	opts.UseColors = true
	opts.ShowTypes = true
	opts.ShowSizes = true
	return opts
}

// CompactOptions returns options for compact output
func CompactOptions() DebugOptions {
	opts := DefaultOptions()
	opts.CompactMode = true
	opts.MaxStringLength = 50
	opts.MaxSliceLength = 5
	return opts
}

// Debug prints a value in debug format (basic version)
func Debug(v any) string {
	return DebugWithOptions(v, DefaultOptions())
}

// Pretty prints a value with colors and extra info
func Pretty(v any) string {
	return DebugWithOptions(v, PrettyOptions())
}

// Compact prints a value in compact format
func Compact(v any) string {
	return DebugWithOptions(v, CompactOptions())
}

// Print functions
func DebugPrint(v any) {
	fmt.Println(Debug(v))
}

func PrettyPrint(v any) {
	fmt.Println(Pretty(v))
}

func CompactPrint(v any) {
	fmt.Println(Compact(v))
}

// Table prints a slice of structs as a table
func Table(slice any) string {
	return TableWithOptions(slice, TableOptions{})
}

func TablePrint(slice any) {
	fmt.Println(Table(slice))
}

// JSON-like output
func JSON(v any) string {
	opts := DefaultOptions()
	opts.CompactMode = false
	return jsonLikeValue(reflect.ValueOf(v), 0, opts)
}

func JSONPrint(v any) {
	fmt.Println(JSON(v))
}

// Diff compares two values and shows differences
func Diff(a, b any) string {
	return diffValues(reflect.ValueOf(a), reflect.ValueOf(b), "")
}

func DiffPrint(a, b any) {
	fmt.Println(Diff(a, b))
}

// Size returns memory size information about a value
func Size(v any) string {
	return sizeInfo(reflect.ValueOf(v))
}

// Hexdump prints binary data in hex format
func Hexdump(data []byte) string {
	return hexdump(data, 16)
}

func HexdumpPrint(data []byte) {
	fmt.Println(Hexdump(data))
}

// Stack prints current stack trace
func Stack() string {
	return stackTrace(10)
}

func StackPrint() {
	fmt.Println(Stack())
}

// Timing utilities
type Timer struct {
	start time.Time
	name  string
}

func StartTimer(name string) *Timer {
	return &Timer{
		start: time.Now(),
		name:  name,
	}
}

func (t *Timer) Stop() string {
	duration := time.Since(t.start)
	return fmt.Sprintf("%s took %v", t.name, duration)
}

func (t *Timer) StopAndPrint() {
	fmt.Println(t.Stop())
}

// Main debug implementation with options
func DebugWithOptions(v any, opts DebugOptions) string {
	return debugValueWithOptions(reflect.ValueOf(v), 0, opts)
}

func debugValueWithOptions(v reflect.Value, depth int, opts DebugOptions) string {
	if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
		return colorize("...", Gray, opts.UseColors)
	}

	if !v.IsValid() {
		return colorize("<invalid>", Red, opts.UseColors)
	}

	// Check for custom formatter
	if opts.CustomFormatters != nil {
		if formatter, exists := opts.CustomFormatters[v.Type()]; exists {
			return formatter(v)
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		return debugStructWithOptions(v, depth, opts)
	case reflect.Ptr:
		return debugPointerWithOptions(v, depth, opts)
	case reflect.Slice, reflect.Array:
		return debugSliceWithOptions(v, depth, opts)
	case reflect.Map:
		return debugMapWithOptions(v, depth, opts)
	case reflect.String:
		return debugStringWithOptions(v, opts)
	case reflect.Chan:
		return debugChanWithOptions(v, opts)
	case reflect.Func:
		return debugFuncWithOptions(v, opts)
	case reflect.Interface:
		if v.IsNil() {
			return colorize("<nil>", Gray, opts.UseColors)
		}
		return debugValueWithOptions(v.Elem(), depth, opts)
	case reflect.Bool:
		color := Green
		if !v.Bool() {
			color = Red
		}
		return colorize(fmt.Sprintf("%v", v.Interface()), color, opts.UseColors)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return colorize(fmt.Sprintf("%d", v.Int()), Cyan, opts.UseColors)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return colorize(fmt.Sprintf("%d", v.Uint()), Cyan, opts.UseColors)
	case reflect.Float32, reflect.Float64:
		return colorize(fmt.Sprintf("%g", v.Float()), Cyan, opts.UseColors)
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func debugStructWithOptions(v reflect.Value, depth int, opts DebugOptions) string {
	t := v.Type()
	var result strings.Builder

	typeName := t.Name()
	if opts.ShowTypes {
		typeName = fmt.Sprintf("%s (%s)", typeName, t.String())
	}
	if opts.ShowSizes {
		typeName = fmt.Sprintf("%s [size: %d]", typeName, t.Size())
	}

	result.WriteString(colorize(typeName, Yellow, opts.UseColors))

	if opts.CompactMode {
		result.WriteString(" { ")
	} else {
		result.WriteString(" {\n")
	}

	fieldCount := 0
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Apply field filter if provided
		if opts.FieldFilter != nil && !opts.FieldFilter(field) {
			continue
		}

		// Skip unexported fields unless ShowPrivate is true
		if !fieldValue.CanInterface() && !opts.ShowPrivate {
			continue
		}

		if fieldCount > 0 && opts.CompactMode {
			result.WriteString(", ")
		}

		if !opts.CompactMode {
			result.WriteString(strings.Repeat(opts.Indent, depth+1))
		}

		fieldName := field.Name
		if opts.ShowTypes {
			fieldName = fmt.Sprintf("%s (%s)", fieldName, field.Type.String())
		}

		result.WriteString(colorize(fieldName, Blue, opts.UseColors))
		result.WriteString(": ")

		if fieldValue.CanInterface() {
			result.WriteString(debugValueWithOptions(fieldValue, depth+1, opts))
		} else {
			result.WriteString(colorize("<unexported>", Gray, opts.UseColors))
		}

		if !opts.CompactMode {
			result.WriteString(",\n")
		}
		fieldCount++
	}

	if opts.CompactMode {
		result.WriteString(" }")
	} else {
		result.WriteString(strings.Repeat(opts.Indent, depth))
		result.WriteString("}")
	}

	return result.String()
}

func debugPointerWithOptions(v reflect.Value, depth int, opts DebugOptions) string {
	if v.IsNil() {
		return colorize("<nil>", Gray, opts.UseColors)
	}

	var result strings.Builder
	result.WriteString("&")

	if opts.ShowAddresses {
		addr := fmt.Sprintf("0x%x", v.Pointer())
		result.WriteString(colorize(fmt.Sprintf("[%s] ", addr), Gray, opts.UseColors))
	}

	result.WriteString(debugValueWithOptions(v.Elem(), depth, opts))
	return result.String()
}

func debugSliceWithOptions(v reflect.Value, depth int, opts DebugOptions) string {
	var result strings.Builder

	length := v.Len()
	capacity := v.Cap()

	prefix := "["
	if opts.ShowSizes {
		if v.Kind() == reflect.Slice {
			prefix = fmt.Sprintf("[len:%d cap:%d ", length, capacity)
		} else {
			prefix = fmt.Sprintf("[len:%d ", length)
		}
	}

	result.WriteString(colorize(prefix, Magenta, opts.UseColors))

	maxLen := length
	truncated := false
	if opts.MaxSliceLength > 0 && length > opts.MaxSliceLength {
		maxLen = opts.MaxSliceLength
		truncated = true
	}

	if opts.CompactMode {
		for i := 0; i < maxLen; i++ {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(debugValueWithOptions(v.Index(i), depth+1, opts))
		}
		if truncated {
			result.WriteString(colorize(fmt.Sprintf(", ... +%d more", length-maxLen), Gray, opts.UseColors))
		}
		result.WriteString(colorize("]", Magenta, opts.UseColors))
	} else {
		if maxLen > 0 {
			result.WriteString("\n")
		}
		for i := 0; i < maxLen; i++ {
			result.WriteString(strings.Repeat(opts.Indent, depth+1))
			result.WriteString(debugValueWithOptions(v.Index(i), depth+1, opts))
			result.WriteString(",\n")
		}
		if truncated {
			result.WriteString(strings.Repeat(opts.Indent, depth+1))
			result.WriteString(colorize(fmt.Sprintf("... +%d more items", length-maxLen), Gray, opts.UseColors))
			result.WriteString("\n")
		}
		result.WriteString(strings.Repeat(opts.Indent, depth))
		result.WriteString(colorize("]", Magenta, opts.UseColors))
	}

	return result.String()
}

func debugMapWithOptions(v reflect.Value, depth int, opts DebugOptions) string {
	var result strings.Builder

	length := v.Len()
	prefix := "{"
	if opts.ShowSizes {
		prefix = fmt.Sprintf("{len:%d ", length)
	}

	result.WriteString(colorize(prefix, Magenta, opts.UseColors))

	keys := v.MapKeys()
	if opts.SortMapKeys {
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
		})
	}

	if opts.CompactMode {
		for i, key := range keys {
			if i > 0 {
				result.WriteString(", ")
			}
			mapValue := v.MapIndex(key)
			result.WriteString(debugValueWithOptions(key, depth+1, opts))
			result.WriteString(": ")
			result.WriteString(debugValueWithOptions(mapValue, depth+1, opts))
		}
		result.WriteString(colorize(" }", Magenta, opts.UseColors))
	} else {
		if len(keys) > 0 {
			result.WriteString("\n")
		}
		for _, key := range keys {
			mapValue := v.MapIndex(key)
			result.WriteString(strings.Repeat(opts.Indent, depth+1))
			result.WriteString(debugValueWithOptions(key, depth+1, opts))
			result.WriteString(": ")
			result.WriteString(debugValueWithOptions(mapValue, depth+1, opts))
			result.WriteString(",\n")
		}
		result.WriteString(strings.Repeat(opts.Indent, depth))
		result.WriteString(colorize("}", Magenta, opts.UseColors))
	}

	return result.String()
}

func debugStringWithOptions(v reflect.Value, opts DebugOptions) string {
	str := v.String()
	length := len(str)

	truncated := false
	if opts.MaxStringLength > 0 && length > opts.MaxStringLength {
		str = str[:opts.MaxStringLength] + "..."
		truncated = true
	}

	result := fmt.Sprintf(`"%s"`, str)
	if opts.ShowSizes || truncated {
		if truncated {
			result = fmt.Sprintf(`"%s" [len:%d, truncated]`, str, length)
		} else {
			result = fmt.Sprintf(`"%s" [len:%d]`, str, length)
		}
	}

	return colorize(result, Green, opts.UseColors)
}

func debugChanWithOptions(v reflect.Value, opts DebugOptions) string {
	if v.IsNil() {
		return colorize("<nil chan>", Gray, opts.UseColors)
	}

	result := fmt.Sprintf("chan(%s)", v.Type().Elem().Name())
	if opts.ShowSizes {
		result = fmt.Sprintf("chan(%s) [cap:%d, len:%d]", v.Type().Elem().Name(), v.Cap(), v.Len())
	}

	return colorize(result, Cyan, opts.UseColors)
}

func debugFuncWithOptions(v reflect.Value, opts DebugOptions) string {
	if v.IsNil() {
		return colorize("<nil func>", Gray, opts.UseColors)
	}

	result := "func"
	if opts.ShowTypes {
		result = fmt.Sprintf("func(%s)", v.Type().String())
	}
	if opts.ShowAddresses {
		result = fmt.Sprintf("%s@0x%x", result, v.Pointer())
	}

	return colorize(result, Cyan, opts.UseColors)
}

// Table formatting for slices of structs
type TableOptions struct {
	MaxColumnWidth int
	ShowTypes      bool
	UseColors      bool
	Separator      string
}

func TableWithOptions(slice any, opts TableOptions) string {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return "Error: not a slice or array"
	}

	if v.Len() == 0 {
		return "Empty slice"
	}

	// Set defaults
	if opts.MaxColumnWidth == 0 {
		opts.MaxColumnWidth = 20
	}
	if opts.Separator == "" {
		opts.Separator = " | "
	}

	first := v.Index(0)
	if first.Kind() != reflect.Struct {
		return "Error: slice elements must be structs"
	}

	return formatTable(v, opts)
}

func formatTable(v reflect.Value, opts TableOptions) string {
	if v.Len() == 0 {
		return ""
	}

	first := v.Index(0)
	t := first.Type()

	// Get field names
	var fields []reflect.StructField
	var headers []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if first.Field(i).CanInterface() {
			fields = append(fields, field)
			header := field.Name
			if opts.ShowTypes {
				header = fmt.Sprintf("%s (%s)", header, field.Type.String())
			}
			headers = append(headers, header)
		}
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	// Get all rows data and calculate max widths
	var rows [][]string
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		row := make([]string, len(fields))

		for j, field := range fields {
			fieldValue := item.FieldByName(field.Name)
			cellValue := fmt.Sprintf("%v", fieldValue.Interface())

			if opts.MaxColumnWidth > 0 && len(cellValue) > opts.MaxColumnWidth {
				cellValue = cellValue[:opts.MaxColumnWidth-3] + "..."
			}

			row[j] = cellValue
			if len(cellValue) > widths[j] {
				widths[j] = len(cellValue)
			}
		}
		rows = append(rows, row)
	}

	var result strings.Builder

	// Header
	for i, header := range headers {
		if i > 0 {
			result.WriteString(opts.Separator)
		}
		formatted := fmt.Sprintf("%-*s", widths[i], header)
		if opts.UseColors {
			formatted = colorize(formatted, Bold+Blue, true)
		}
		result.WriteString(formatted)
	}
	result.WriteString("\n")

	// Separator line
	for i, width := range widths {
		if i > 0 {
			result.WriteString(strings.Repeat("-", len(opts.Separator)))
		}
		result.WriteString(strings.Repeat("-", width))
	}
	result.WriteString("\n")

	// Data rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				result.WriteString(opts.Separator)
			}
			result.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// JSON-like formatting
func jsonLikeValue(v reflect.Value, depth int, opts DebugOptions) string {
	if !v.IsValid() {
		return "null"
	}

	switch v.Kind() {
	case reflect.Struct:
		return jsonLikeStruct(v, depth, opts)
	case reflect.Map:
		return jsonLikeMap(v, depth, opts)
	case reflect.Slice, reflect.Array:
		return jsonLikeSlice(v, depth, opts)
	case reflect.String:
		return fmt.Sprintf(`"%s"`, v.String())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Ptr:
		if v.IsNil() {
			return "null"
		}
		return jsonLikeValue(v.Elem(), depth, opts)
	case reflect.Interface:
		if v.IsNil() {
			return "null"
		}
		return jsonLikeValue(v.Elem(), depth, opts)
	default:
		return fmt.Sprintf(`"%v"`, v.Interface())
	}
}

func jsonLikeStruct(v reflect.Value, depth int, opts DebugOptions) string {
	var result strings.Builder
	result.WriteString("{\n")

	t := v.Type()
	fieldCount := 0

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanInterface() && !opts.ShowPrivate {
			continue
		}

		if fieldCount > 0 {
			result.WriteString(",\n")
		}

		result.WriteString(strings.Repeat(opts.Indent, depth+1))
		result.WriteString(fmt.Sprintf(`"%s": `, field.Name))

		if fieldValue.CanInterface() {
			result.WriteString(jsonLikeValue(fieldValue, depth+1, opts))
		} else {
			result.WriteString("null")
		}

		fieldCount++
	}

	result.WriteString("\n")
	result.WriteString(strings.Repeat(opts.Indent, depth))
	result.WriteString("}")

	return result.String()
}

func jsonLikeMap(v reflect.Value, depth int, opts DebugOptions) string {
	var result strings.Builder
	result.WriteString("{\n")

	keys := v.MapKeys()
	if opts.SortMapKeys {
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
		})
	}

	for i, key := range keys {
		if i > 0 {
			result.WriteString(",\n")
		}

		mapValue := v.MapIndex(key)
		result.WriteString(strings.Repeat(opts.Indent, depth+1))
		result.WriteString(fmt.Sprintf(`"%v": `, key.Interface()))
		result.WriteString(jsonLikeValue(mapValue, depth+1, opts))
	}

	result.WriteString("\n")
	result.WriteString(strings.Repeat(opts.Indent, depth))
	result.WriteString("}")

	return result.String()
}

func jsonLikeSlice(v reflect.Value, depth int, opts DebugOptions) string {
	var result strings.Builder
	result.WriteString("[\n")

	length := v.Len()
	maxLen := length
	if opts.MaxSliceLength > 0 && length > opts.MaxSliceLength {
		maxLen = opts.MaxSliceLength
	}

	for i := 0; i < maxLen; i++ {
		if i > 0 {
			result.WriteString(",\n")
		}

		result.WriteString(strings.Repeat(opts.Indent, depth+1))
		result.WriteString(jsonLikeValue(v.Index(i), depth+1, opts))
	}

	result.WriteString("\n")
	result.WriteString(strings.Repeat(opts.Indent, depth))
	result.WriteString("]")

	return result.String()
}

// Diff functionality
func diffValues(a, b reflect.Value, path string) string {
	var result strings.Builder

	if !a.IsValid() && !b.IsValid() {
		return ""
	}

	if !a.IsValid() {
		result.WriteString(fmt.Sprintf("- %s: <missing>\n", path))
		result.WriteString(fmt.Sprintf("+ %s: %v\n", path, b.Interface()))
		return result.String()
	}

	if !b.IsValid() {
		result.WriteString(fmt.Sprintf("- %s: %v\n", path, a.Interface()))
		result.WriteString(fmt.Sprintf("+ %s: <missing>\n", path))
		return result.String()
	}

	if a.Type() != b.Type() {
		result.WriteString(fmt.Sprintf("- %s: %v (%s)\n", path, a.Interface(), a.Type()))
		result.WriteString(fmt.Sprintf("+ %s: %v (%s)\n", path, b.Interface(), b.Type()))
		return result.String()
	}

	switch a.Kind() {
	case reflect.Struct:
		return diffStructs(a, b, path)
	case reflect.Slice, reflect.Array:
		return diffSlices(a, b, path)
	case reflect.Map:
		return diffMaps(a, b, path)
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			result.WriteString(fmt.Sprintf("- %s: %v\n", path, a.Interface()))
			result.WriteString(fmt.Sprintf("+ %s: %v\n", path, b.Interface()))
		}
	}

	return result.String()
}

func diffStructs(a, b reflect.Value, path string) string {
	var result strings.Builder
	t := a.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldPath := path + "." + field.Name
		if path == "" {
			fieldPath = field.Name
		}

		aField := a.Field(i)
		bField := b.Field(i)

		if aField.CanInterface() && bField.CanInterface() {
			result.WriteString(diffValues(aField, bField, fieldPath))
		}
	}

	return result.String()
}

func diffSlices(a, b reflect.Value, path string) string {
	var result strings.Builder

	aLen := a.Len()
	bLen := b.Len()

	minLen := aLen
	if bLen < minLen {
		minLen = bLen
	}

	for i := 0; i < minLen; i++ {
		indexPath := fmt.Sprintf("%s[%d]", path, i)
		result.WriteString(diffValues(a.Index(i), b.Index(i), indexPath))
	}

	if aLen != bLen {
		result.WriteString(fmt.Sprintf("~ %s: length differs (was %d, now %d)\n", path, aLen, bLen))

		if aLen > bLen {
			for i := bLen; i < aLen; i++ {
				indexPath := fmt.Sprintf("%s[%d]", path, i)
				result.WriteString(fmt.Sprintf("- %s: %v\n", indexPath, a.Index(i).Interface()))
			}
		} else {
			for i := aLen; i < bLen; i++ {
				indexPath := fmt.Sprintf("%s[%d]", path, i)
				result.WriteString(fmt.Sprintf("+ %s: %v\n", indexPath, b.Index(i).Interface()))
			}
		}
	}

	return result.String()
}

func diffMaps(a, b reflect.Value, path string) string {
	var result strings.Builder

	aKeys := make(map[any]bool)
	for _, key := range a.MapKeys() {
		aKeys[key.Interface()] = true
	}

	bKeys := make(map[any]bool)
	for _, key := range b.MapKeys() {
		bKeys[key.Interface()] = true
	}

	// Check common keys
	for _, key := range a.MapKeys() {
		keyInterface := key.Interface()
		if bKeys[keyInterface] {
			keyPath := fmt.Sprintf("%s[%v]", path, keyInterface)
			aValue := a.MapIndex(key)
			bValue := b.MapIndex(key)
			result.WriteString(diffValues(aValue, bValue, keyPath))
		} else {
			keyPath := fmt.Sprintf("%s[%v]", path, keyInterface)
			result.WriteString(fmt.Sprintf("- %s: %v\n", keyPath, a.MapIndex(key).Interface()))
		}
	}

	// Check keys only in b
	for _, key := range b.MapKeys() {
		keyInterface := key.Interface()
		if !aKeys[keyInterface] {
			keyPath := fmt.Sprintf("%s[%v]", path, keyInterface)
			result.WriteString(fmt.Sprintf("+ %s: %v\n", keyPath, b.MapIndex(key).Interface()))
		}
	}

	return result.String()
}

// Utility functions
func colorize(text, color string, useColors bool) string {
	if !useColors {
		return text
	}
	return color + text + Reset
}

func sizeInfo(v reflect.Value) string {
	if !v.IsValid() {
		return "invalid value"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Type: %s\n", v.Type()))
	result.WriteString(fmt.Sprintf("Kind: %s\n", v.Kind()))
	result.WriteString(fmt.Sprintf("Type size: %d bytes\n", v.Type().Size()))

	switch v.Kind() {
	case reflect.String:
		result.WriteString(fmt.Sprintf("String length: %d\n", v.Len()))
	case reflect.Slice:
		result.WriteString(fmt.Sprintf("Slice length: %d\n", v.Len()))
		result.WriteString(fmt.Sprintf("Slice capacity: %d\n", v.Cap()))
		if v.Len() > 0 {
			elemSize := v.Type().Elem().Size()
			result.WriteString(fmt.Sprintf("Element size: %d bytes\n", elemSize))
			result.WriteString(fmt.Sprintf("Total size: ~%d bytes\n", uintptr(v.Len())*elemSize))
		}
	case reflect.Array:
		result.WriteString(fmt.Sprintf("Array length: %d\n", v.Len()))
		elemSize := v.Type().Elem().Size()
		result.WriteString(fmt.Sprintf("Element size: %d bytes\n", elemSize))
		result.WriteString(fmt.Sprintf("Total size: %d bytes\n", uintptr(v.Len())*elemSize))
	case reflect.Map:
		result.WriteString(fmt.Sprintf("Map length: %d\n", v.Len()))
	case reflect.Chan:
		result.WriteString(fmt.Sprintf("Channel length: %d\n", v.Len()))
		result.WriteString(fmt.Sprintf("Channel capacity: %d\n", v.Cap()))
	case reflect.Ptr:
		if !v.IsNil() {
			result.WriteString(fmt.Sprintf("Pointer address: 0x%x\n", v.Pointer()))
			result.WriteString("Points to:\n")
			result.WriteString(sizeInfo(v.Elem()))
		}
	case reflect.Struct:
		result.WriteString(fmt.Sprintf("Number of fields: %d\n", v.NumField()))
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			result.WriteString(fmt.Sprintf("  %s: %d bytes\n", field.Name, field.Type.Size()))
		}
	}

	return result.String()
}

func hexdump(data []byte, width int) string {
	var result strings.Builder

	for i := 0; i < len(data); i += width {
		// Address
		result.WriteString(fmt.Sprintf("%08x  ", i))

		// Hex bytes
		for j := 0; j < width; j++ {
			if i+j < len(data) {
				result.WriteString(fmt.Sprintf("%02x ", data[i+j]))
			} else {
				result.WriteString("   ")
			}

			if j == width/2-1 {
				result.WriteString(" ")
			}
		}

		result.WriteString(" |")

		// ASCII representation
		for j := 0; j < width && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				result.WriteString(string(b))
			} else {
				result.WriteString(".")
			}
		}

		result.WriteString("|\n")
	}

	return result.String()
}

func stackTrace(maxFrames int) string {
	var result strings.Builder

	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(2, pc) // Skip stackTrace and runtime.Callers
	frames := runtime.CallersFrames(pc[:n])

	for i := 0; ; i++ {
		frame, more := frames.Next()
		if !more {
			break
		}

		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, frame.Function))
		result.WriteString(fmt.Sprintf("   %s:%d\n", frame.File, frame.Line))

		if !more {
			break
		}
	}

	return result.String()
}

// Advanced formatters
func WithCustomFormatter(t reflect.Type, formatter func(reflect.Value) string) DebugOptions {
	opts := DefaultOptions()
	opts.CustomFormatters = make(map[reflect.Type]func(reflect.Value) string)
	opts.CustomFormatters[t] = formatter
	return opts
}

func WithFieldFilter(filter func(reflect.StructField) bool) DebugOptions {
	opts := DefaultOptions()
	opts.FieldFilter = filter
	return opts
}

// Convenience functions for common filters
func PublicFieldsOnly(field reflect.StructField) bool {
	return field.IsExported()
}

func FieldsWithTag(tag string) func(reflect.StructField) bool {
	return func(field reflect.StructField) bool {
		return field.Tag.Get(tag) != ""
	}
}

func FieldsWithoutTag(tag string) func(reflect.StructField) bool {
	return func(field reflect.StructField) bool {
		return field.Tag.Get(tag) == ""
	}
}

func FieldNameMatches(pattern string) func(reflect.StructField) bool {
	regex := regexp.MustCompile(pattern)
	return func(field reflect.StructField) bool {
		return regex.MatchString(field.Name)
	}
}

// Type-specific formatters
func TimeFormatter(v reflect.Value) string {
	if t, ok := v.Interface().(time.Time); ok {
		return fmt.Sprintf(`"%s"`, t.Format(time.RFC3339))
	}
	return fmt.Sprintf("%v", v.Interface())
}

func DurationFormatter(v reflect.Value) string {
	if d, ok := v.Interface().(time.Duration); ok {
		return fmt.Sprintf(`"%s"`, d.String())
	}
	return fmt.Sprintf("%v", v.Interface())
}

// Memory utilities
func MemoryAddress(v any) string {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		return fmt.Sprintf("0x%x", value.Pointer())
	}
	if value.CanAddr() {
		return fmt.Sprintf("0x%x", value.UnsafeAddr())
	}
	return "not addressable"
}

func UnsafeString(v any) string {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.String {
		str := value.String()
		header := (*reflect.StringHeader)(unsafe.Pointer(&str))
		return fmt.Sprintf("String{Data: 0x%x, Len: %d}", header.Data, header.Len)
	}
	return "not a string"
}

func UnsafeSlice(v any) string {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Slice {
		return "not a slice"
	}

	// Check if we can get the slice header safely
	if value.Len() == 0 {
		return "Slice{Data: <nil>, Len: 0, Cap: 0}"
	}

	// For non-addressable slices, we can still get some information
	if !value.CanAddr() {
		return fmt.Sprintf("Slice{Len: %d, Cap: %d} (unaddressable)", value.Len(), value.Cap())
	}

	// Safe to get the slice header
	header := (*reflect.SliceHeader)(unsafe.Pointer(value.UnsafeAddr()))
	return fmt.Sprintf("Slice{Data: 0x%x, Len: %d, Cap: %d}", header.Data, header.Len, header.Cap)
}

// Better version that works with both addressable and unaddressable slices
func UnsafeSliceInfo(v any) string {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Slice {
		return "not a slice"
	}

	result := fmt.Sprintf("Slice{Len: %d, Cap: %d", value.Len(), value.Cap())

	// Try to get data pointer if possible
	if value.Len() > 0 {
		firstElem := value.Index(0)
		if firstElem.CanAddr() {
			dataPtr := firstElem.UnsafeAddr()
			result += fmt.Sprintf(", Data: 0x%x", dataPtr)
		} else {
			result += ", Data: <unaddressable>"
		}
	} else {
		result += ", Data: <nil>"
	}

	result += "}"
	return result
}
