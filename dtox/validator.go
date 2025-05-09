package dtox

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Abraxas-365/craftale/errx"
)

// ValidationRule defines a validation rule for a specific field
type ValidationRule struct {
	FieldName string
	Validator func(value interface{}) error
	Message   string
}

// ValidationError represents an error that occurred during validation
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface for ValidationError
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

// ValidationErrors holds multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	if len(e) == 1 {
		return e[0].Error()
	}

	return fmt.Sprintf("%d validation errors occurred", len(e))
}

// ToErrx converts ValidationErrors to an errx.Error
func (e ValidationErrors) ToErrx() *errx.Error {
	if len(e) == 0 {
		return nil
	}

	// Create detailed error information
	details := make(map[string]any)
	for i, err := range e {
		details[fmt.Sprintf("field_%d", i)] = map[string]string{
			"field":   err.Field,
			"message": err.Message,
		}
	}

	return ErrorRegistry.New(ErrValidationFailed).
		WithDetails(details)
}

// WithRules adds multiple validation rules to the mapper
func (m *Mapper[TDto, TModel]) WithRules(rules []ValidationRule) *Mapper[TDto, TModel] {
	if len(rules) == 0 {
		return m
	}

	// Create a validation function that applies all the rules
	m.validationFn = func(dto TDto) error {
		val := reflect.ValueOf(dto)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Process all rules
		var validationErrors ValidationErrors

		for _, rule := range rules {
			fieldVal := val.FieldByName(rule.FieldName)
			if !fieldVal.IsValid() {
				// Skip invalid fields
				continue
			}

			var fieldInterface interface{}
			if fieldVal.CanInterface() {
				fieldInterface = fieldVal.Interface()
			} else {
				// Skip fields we can't get an interface for
				continue
			}

			if err := rule.Validator(fieldInterface); err != nil {
				message := rule.Message
				if message == "" {
					message = err.Error()
				}

				validationErrors = append(validationErrors, ValidationError{
					Field:   rule.FieldName,
					Message: message,
				})
			}
		}

		if len(validationErrors) > 0 {
			return validationErrors
		}

		return nil
	}

	return m
}

// Common validation functions

// Required checks if a value is not empty
func Required(value interface{}) error {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return errors.New("field is required")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if v.Len() == 0 {
			return errors.New("field is required")
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return errors.New("field is required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() == 0 {
			return errors.New("field is required")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() == 0 {
			return errors.New("field is required")
		}
	case reflect.Float32, reflect.Float64:
		if v.Float() == 0 {
			return errors.New("field is required")
		}
	case reflect.Bool:
		// For bool, we consider it "filled" regardless of value
		return nil
	default:
		return errors.New("cannot check if field is required")
	}

	return nil
}

// MinLength checks if a string has at least the specified length
func MinLength(min int) func(interface{}) error {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return errors.New("value is not a string")
		}

		if len(s) < min {
			return fmt.Errorf("must be at least %d characters", min)
		}

		return nil
	}
}

// MaxLength checks if a string is at most the specified length
func MaxLength(max int) func(interface{}) error {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return errors.New("value is not a string")
		}

		if len(s) > max {
			return fmt.Errorf("must be at most %d characters", max)
		}

		return nil
	}
}

// MinValue checks if a numeric value is at least the specified minimum
func MinValue(min float64) func(interface{}) error {
	return func(value interface{}) error {
		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) < min {
				return fmt.Errorf("must be at least %v", min)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) < min {
				return fmt.Errorf("must be at least %v", min)
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() < min {
				return fmt.Errorf("must be at least %v", min)
			}
		default:
			return errors.New("value is not numeric")
		}

		return nil
	}
}

// MaxValue checks if a numeric value is at most the specified maximum
func MaxValue(max float64) func(interface{}) error {
	return func(value interface{}) error {
		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) > max {
				return fmt.Errorf("must be at most %v", max)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) > max {
				return fmt.Errorf("must be at most %v", max)
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() > max {
				return fmt.Errorf("must be at most %v", max)
			}
		default:
			return errors.New("value is not numeric")
		}

		return nil
	}
}
