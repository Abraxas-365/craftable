package validatex

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Abraxas-365/craftable/errx"
)

// Error registry for validatex
var (
	ValidatorErrors = errx.NewRegistry("VALIDATOR")

	// Common validation error codes
	ErrValidationFailed  = ValidatorErrors.Register("VALIDATION_FAILED", errx.TypeValidation, http.StatusBadRequest, "Validation failed")
	ErrRequiredField     = ValidatorErrors.Register("REQUIRED_FIELD", errx.TypeValidation, http.StatusBadRequest, "Field is required")
	ErrInvalidEmail      = ValidatorErrors.Register("INVALID_EMAIL", errx.TypeValidation, http.StatusBadRequest, "Invalid email format")
	ErrInvalidURL        = ValidatorErrors.Register("INVALID_URL", errx.TypeValidation, http.StatusBadRequest, "Invalid URL format")
	ErrBelowMin          = ValidatorErrors.Register("BELOW_MIN", errx.TypeValidation, http.StatusBadRequest, "Value below minimum")
	ErrAboveMax          = ValidatorErrors.Register("ABOVE_MAX", errx.TypeValidation, http.StatusBadRequest, "Value above maximum")
	ErrInvalidOption     = ValidatorErrors.Register("INVALID_OPTION", errx.TypeValidation, http.StatusBadRequest, "Invalid option")
	ErrPatternMismatch   = ValidatorErrors.Register("PATTERN_MISMATCH", errx.TypeValidation, http.StatusBadRequest, "Value doesn't match pattern")
	ErrInvalidUUID       = ValidatorErrors.Register("INVALID_UUID", errx.TypeValidation, http.StatusBadRequest, "Invalid UUID format")
	ErrInvalidAlphaNum   = ValidatorErrors.Register("INVALID_ALPHANUM", errx.TypeValidation, http.StatusBadRequest, "Value contains non-alphanumeric characters")
	ErrInvalidAlpha      = ValidatorErrors.Register("INVALID_ALPHA", errx.TypeValidation, http.StatusBadRequest, "Value contains non-alphabetic characters")
	ErrInvalidNumeric    = ValidatorErrors.Register("INVALID_NUMERIC", errx.TypeValidation, http.StatusBadRequest, "Value contains non-numeric characters")
	ErrUnknownValidator  = ValidatorErrors.Register("UNKNOWN_VALIDATOR", errx.TypeInternal, http.StatusInternalServerError, "Unknown validator")
	ErrInvalidStruct     = ValidatorErrors.Register("INVALID_STRUCT", errx.TypeBadRequest, http.StatusBadRequest, "Value is not a struct")
	ErrUnsupportedType   = ValidatorErrors.Register("UNSUPPORTED_TYPE", errx.TypeInternal, http.StatusInternalServerError, "Unsupported type")
	ErrInvalidValidation = ValidatorErrors.Register("INVALID_VALIDATION", errx.TypeInternal, http.StatusInternalServerError, "Invalid validation rule")
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string // Field name
	Rule    string // Rule that failed
	Param   string // Rule parameter (if any)
	Value   any    // Value that was validated
	Message string // Error message
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a slice of ValidationError
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	if len(e) == 1 {
		return e[0].Error()
	}

	errMsgs := make([]string, len(e))
	for i, err := range e {
		errMsgs[i] = fmt.Sprintf("  - %s", err.Error())
	}

	return fmt.Sprintf("%d validation errors:\n%s", len(e), strings.Join(errMsgs, "\n"))
}

// Has checks if the validation errors contain an error for the specified field
func (e ValidationErrors) Has(field string) bool {
	for _, err := range e {
		if err.Field == field {
			return true
		}
	}
	return false
}

// Get returns all validation errors for the specified field
func (e ValidationErrors) Get(field string) []ValidationError {
	var result []ValidationError
	for _, err := range e {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}

// ByRule returns all validation errors for the specified rule
func (e ValidationErrors) ByRule(rule string) []ValidationError {
	var result []ValidationError
	for _, err := range e {
		if err.Rule == rule {
			result = append(result, err)
		}
	}
	return result
}

// ToErrx converts ValidationErrors to an errx.Error
func (e ValidationErrors) ToErrx() *errx.Error {
	if len(e) == 0 {
		return nil
	}

	// Create the main error
	errx := ValidatorErrors.New(ErrValidationFailed)

	// Create structured details about the validation errors
	fieldErrors := make(map[string][]map[string]any)

	for _, ve := range e {
		// Extract all error information for this field
		errorInfo := map[string]any{
			"rule":    ve.Rule,
			"message": ve.Message,
		}

		if ve.Param != "" {
			errorInfo["param"] = ve.Param
		}

		// Use string representation of the value to avoid JSON serialization issues
		errorInfo["value"] = fmt.Sprintf("%v", ve.Value)

		// Add to the map of field errors, creating the slice if it doesn't exist
		if _, ok := fieldErrors[ve.Field]; !ok {
			fieldErrors[ve.Field] = make([]map[string]any, 0)
		}
		fieldErrors[ve.Field] = append(fieldErrors[ve.Field], errorInfo)
	}

	// Add overall details
	details := map[string]any{
		"errors":      fieldErrors,
		"error_count": len(e),
		"field_count": len(fieldErrors),
	}

	return errx.WithDetails(details)
}

// NewValidationError creates a new validation error
func NewValidationError(field, rule, param string, value any, message string) ValidationError {
	if message == "" {
		message = getErrorMessageForRule(rule, param)
	}

	return ValidationError{
		Field:   field,
		Rule:    rule,
		Param:   param,
		Value:   value,
		Message: message,
	}
}

var customErrorMessages = make(map[string]string)

// SetCustomErrorMessage sets a custom error message for a validation rule
func SetCustomErrorMessage(rule, message string) {
	customErrorMessages[rule] = message
}

// getErrorMessageForRule returns a default error message for a rule
func getErrorMessageForRule(rule, param string) string {
	if msg, ok := customErrorMessages[rule]; ok {
		return msg
	}
	switch rule {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "oneof":
		return fmt.Sprintf("must be one of: %s", param)
	case "regex":
		return "must match the required pattern"
	case "uuid":
		return "must be a valid UUID"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "alpha":
		return "must contain only alphabetic characters"
	case "numeric":
		return "must contain only numeric characters"
	default:
		return "failed validation"
	}
}

// getErrorCodeForRule maps a validation rule to an errx Code
func getErrorCodeForRule(rule string) errx.Code {
	switch rule {
	case "required":
		return ErrRequiredField
	case "email":
		return ErrInvalidEmail
	case "url":
		return ErrInvalidURL
	case "min":
		return ErrBelowMin
	case "max":
		return ErrAboveMax
	case "oneof":
		return ErrInvalidOption
	case "regex":
		return ErrPatternMismatch
	case "uuid":
		return ErrInvalidUUID
	case "alphanum":
		return ErrInvalidAlphaNum
	case "alpha":
		return ErrInvalidAlpha
	case "numeric":
		return ErrInvalidNumeric
	default:
		return ErrValidationFailed
	}
}
