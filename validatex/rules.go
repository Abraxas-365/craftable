package validatex

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
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

// validateRequired validates that a value is not empty
func validateRequired(value any, _ string) bool {
	return !isZero(value)
}

// validateEmail validates that a value is a valid email address
func validateEmail(value any, _ string) bool {
	if str, ok := value.(string); ok {
		_, err := mail.ParseAddress(str)
		return err == nil
	}
	return false
}

// validateURL validates that a value is a valid URL
func validateURL(value any, _ string) bool {
	if str, ok := value.(string); ok {
		_, err := url.ParseRequestURI(str)
		return err == nil && strings.Contains(str, ".")
	}
	return false
}

// validateMin validates that a value is at least a minimum
func validateMin(value any, param string) bool {
	min, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	switch v := value.(type) {
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
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Array {
			return rv.Len() >= min
		}
		return false
	}
}

// validateMax validates that a value is at most a maximum
func validateMax(value any, param string) bool {
	max, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	switch v := value.(type) {
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
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Array {
			return rv.Len() <= max
		}
		return false
	}
}

// validateOneOf validates that a value is one of a list of values
func validateOneOf(value any, param string) bool {
	allowedValues := strings.Fields(param)
	if len(allowedValues) == 0 {
		return false
	}

	strValue := fmt.Sprintf("%v", value)
	for _, v := range allowedValues {
		if v == strValue {
			return true
		}
	}

	return false
}

// validateRegex validates that a value matches a regular expression
func validateRegex(value any, param string) bool {
	if str, ok := value.(string); ok {
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
	if str, ok := value.(string); ok {
		return uuidRegex.MatchString(strings.ToLower(str))
	}
	return false
}

// validateAlphaNum validates that a value contains only alphanumeric characters
func validateAlphaNum(value any, _ string) bool {
	if str, ok := value.(string); ok {
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
	if str, ok := value.(string); ok {
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
	if str, ok := value.(string); ok {
		for _, char := range str {
			if !unicode.IsNumber(char) {
				return false
			}
		}
		return true
	}
	return false
}
