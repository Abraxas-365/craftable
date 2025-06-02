package validatex

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// ValidationFunc defines a function that validates a value
type ValidationFunc func(value any, param string) bool

// builtinValidationFuncs is a map of built-in validation functions
var builtinValidationFuncs = map[string]ValidationFunc{
	"required": validateRequired,
	"email":    validateEmail,
	"url":      validateURL,
	"min":      validateMin,
	"max":      validateMax,
	"oneof":    validateOneOf,
	"regex":    validateRegex,
	"uuid":     validateUUID,
	"alphanum": validateAlphaNum,
	"alpha":    validateAlpha,
	"numeric":  validateNumeric,
}

// customValidationFuncs is a map of user-registered validation functions
var customValidationFuncs = map[string]ValidationFunc{}

// RegisterValidationFunc registers a custom validation function
func RegisterValidationFunc(name string, fn ValidationFunc) {
	customValidationFuncs[name] = fn
}

// getValidationFunc returns a validation function by name
func getValidationFunc(name string) (ValidationFunc, bool) {
	// Check custom functions first
	if fn, ok := customValidationFuncs[name]; ok {
		return fn, true
	}

	// Check built-in functions
	if fn, ok := builtinValidationFuncs[name]; ok {
		return fn, true
	}

	return nil, false
}

// dereferenceForValidation safely dereferences a pointer for validation
// Returns the dereferenced value and whether the original was nil
func dereferenceForValidation(value any) (any, bool) {
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

// validateRequired validates that a value is not empty
func validateRequired(value any, _ string) bool {
	if value == nil {
		return false
	}

	val := reflect.ValueOf(value)

	// For pointers, check if they're nil
	if val.Kind() == reflect.Ptr {
		return !val.IsNil()
	}

	// For non-pointers, use the existing isZero logic
	return !isZero(value)
}

// validateEmail validates that a value is a valid email address
func validateEmail(value any, _ string) bool {
	// Dereference if it's a pointer
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true // nil pointers are valid for optional fields
	}

	if str, ok := actualValue.(string); ok {
		_, err := mail.ParseAddress(str)
		return err == nil
	}
	return false
}

// validateURL validates that a value is a valid URL
func validateURL(value any, _ string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		_, err := url.ParseRequestURI(str)
		return err == nil && strings.Contains(str, ".")
	}
	return false
}

// validateMin validates that a value is at least a minimum
func validateMin(value any, param string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	min, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	switch v := actualValue.(type) {
	case string:
		return len(v) >= min
	case int:
		return v >= min
	case int8:
		return int(v) >= min
	case int16:
		return int(v) >= min
	case int32:
		return int(v) >= min
	case int64:
		return int(v) >= min
	case uint:
		return int(v) >= min
	case uint8:
		return int(v) >= min
	case uint16:
		return int(v) >= min
	case uint32:
		return int(v) >= min
	case uint64:
		return int(v) >= min
	case float32:
		return float64(v) >= float64(min)
	case float64:
		return v >= float64(min)
	default:
		// For slices, maps, arrays, check length
		rv := reflect.ValueOf(actualValue)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Array {
			return rv.Len() >= min
		}
		return false
	}
}

// validateMax validates that a value is at most a maximum
func validateMax(value any, param string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	max, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	switch v := actualValue.(type) {
	case string:
		return len(v) <= max
	case int:
		return v <= max
	case int8:
		return int(v) <= max
	case int16:
		return int(v) <= max
	case int32:
		return int(v) <= max
	case int64:
		return int(v) <= max
	case uint:
		return int(v) <= max
	case uint8:
		return int(v) <= max
	case uint16:
		return int(v) <= max
	case uint32:
		return int(v) <= max
	case uint64:
		return int(v) <= max
	case float32:
		return float64(v) <= float64(max)
	case float64:
		return v <= float64(max)
	default:
		// For slices, maps, arrays, check length
		rv := reflect.ValueOf(actualValue)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Array {
			return rv.Len() <= max
		}
		return false
	}
}

// validateOneOf validates that a value is one of a list of values
func validateOneOf(value any, param string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	allowedValues := strings.Fields(param)
	if len(allowedValues) == 0 {
		return false
	}

	strValue := fmt.Sprintf("%v", actualValue)
	return slices.Contains(allowedValues, strValue)
}

// validateRegex validates that a value matches a regular expression
func validateRegex(value any, param string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		re, err := regexp.Compile(param)
		if err != nil {
			return false
		}
		return re.MatchString(str)
	}
	return false
}

// validateUUID validates that a value is a valid UUID
var uuidRegex = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func validateUUID(value any, _ string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		return uuidRegex.MatchString(strings.ToLower(str))
	}
	return false
}

// validateAlphaNum validates that a value contains only alphanumeric characters
func validateAlphaNum(value any, _ string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		for _, char := range str {
			if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
				return false
			}
		}
		return true
	}
	return false
}

// validateAlpha validates that a value contains only alphabetic characters
func validateAlpha(value any, _ string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		for _, char := range str {
			if !unicode.IsLetter(char) {
				return false
			}
		}
		return true
	}
	return false
}

// validateNumeric validates that a value contains only numeric characters
func validateNumeric(value any, _ string) bool {
	actualValue, isNil := dereferenceForValidation(value)
	if isNil {
		return true
	}

	if str, ok := actualValue.(string); ok {
		for _, char := range str {
			if !unicode.IsNumber(char) {
				return false
			}
		}
		return true
	}
	return false
}

