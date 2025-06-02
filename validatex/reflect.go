package validatex

import (
	"errors"
	"reflect"
	"strings"
)

var (
	ErrNotStruct = errors.New("value must be a struct")
)

// structFields returns a map of field names to field values for a struct
func structFields(obj any) (map[string]fieldInfo, error) {
	val := reflect.ValueOf(obj)

	// If obj is a pointer, dereference it
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	// Only process structs
	if val.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	typ := val.Type()
	fields := make(map[string]fieldInfo)

	// Process all fields in the struct
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the validatex tag
		tag := field.Tag.Get("validatex")
		if tag == "" || tag == "-" {
			continue
		}

		// Get the field value
		fieldValue := val.Field(i)
		fieldInterface := fieldValue.Interface()

		// Add to the map - store the actual interface{} value (including pointers)
		fields[field.Name] = fieldInfo{
			Name:  field.Name,
			Value: fieldInterface, // This preserves pointer types
			Rules: parseTag(tag),
			Type:  field.Type,
		}

		// For struct recursion, dereference pointers
		actualFieldValue := fieldValue
		if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
			actualFieldValue = fieldValue.Elem()
		}

		// If the field is a struct, recursively process it
		if actualFieldValue.Kind() == reflect.Struct {
			// Check if the struct implements Validatable
			if _, ok := actualFieldValue.Interface().(Validatable); ok {
				// Already handled by the Validatable interface
				continue
			}

			// Check for embedded struct
			nestedFields, err := structFields(actualFieldValue.Interface())
			if err != nil {
				return nil, err
			}

			// Add nested fields with prefix
			for k, v := range nestedFields {
				fields[field.Name+"."+k] = v
			}
		}
	}

	return fields, nil
}

// fieldInfo stores information about a struct field
type fieldInfo struct {
	Name  string
	Value any
	Rules []ruleInfo
	Type  reflect.Type
}

// ruleInfo stores information about a validation rule
type ruleInfo struct {
	Name  string
	Param string
}

// parseTag parses a validatex tag string into validation rules
func parseTag(tag string) []ruleInfo {
	if tag == "" {
		return nil
	}

	parts := strings.Split(tag, ",")
	rules := make([]ruleInfo, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by equals if present
		nameParam := strings.SplitN(part, "=", 2)
		name := nameParam[0]

		// Get parameter if present
		param := ""
		if len(nameParam) > 1 {
			param = nameParam[1]
		}

		rules = append(rules, ruleInfo{
			Name:  name,
			Param: param,
		})
	}

	return rules
}

// isZero checks if a value is the zero value for its type
func isZero(value any) bool {
	if value == nil {
		return true
	}

	val := reflect.ValueOf(value)

	// For pointers, nil is considered zero (optional field)
	if val.Kind() == reflect.Ptr {
		return val.IsNil()
	}

	// Check for zero value based on type
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	default:
		return reflect.DeepEqual(val.Interface(), reflect.Zero(val.Type()).Interface())
	}
}

// dereferenceValue safely dereferences a pointer value
// Returns the dereferenced value and whether it was nil
func dereferenceValue(value any) (any, bool) {
	if value == nil {
		return nil, true
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return value, false
	}

	if val.IsNil() {
		return nil, true
	}

	return val.Elem().Interface(), false
}
