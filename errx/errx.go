package errx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Code represents a unique error code for each type of error
type Code string

// Type represents the general category of the error
type Type string

const (
	// Error Types
	TypeValidation    Type = "VALIDATION"
	TypeAuthorization Type = "AUTHORIZATION"
	TypeNotFound      Type = "NOT_FOUND"
	TypeConflict      Type = "CONFLICT"
	TypeInternal      Type = "INTERNAL"
	TypeBadRequest    Type = "BAD_REQUEST"
	TypeRateLimit     Type = "RATE_LIMIT"
	TypeBusiness      Type = "BUSINESS"    // For business logic errors
	TypeSystem        Type = "SYSTEM"      // For system/infrastructure errors
	TypeExternal      Type = "EXTERNAL"    // For external service errors
	TypeTimeout       Type = "TIMEOUT"     // For timeout errors
	TypeUnavailable   Type = "UNAVAILABLE" // For service unavailability
)

// Error represents a standardized error
type Error struct {
	Code       Code           `json:"code"`
	Type       Type           `json:"type"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	HTTPStatus int            `json:"-"` // Not exposed in JSON
	cause      error          `json:"-"` // Underlying cause (not serialized)
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

func Print(e error) string {
	if e == nil {
		return "nil"
	}

	var xerr *Error
	if errors.As(e, &xerr) {
		var details string
		if xerr.Details != nil {
			details = ""
			for key, value := range xerr.Details {
				details += fmt.Sprintf("%s: %v, ", key, value)
			}
		}

		if details != "" {
			return fmt.Sprintf("Error: %s, Details: {%s}, HTTP Status: %d", xerr.Error(), details, xerr.HTTPStatus)
		}

		return fmt.Sprintf("Error: %s, HTTP Status: %d", xerr.Error(), xerr.HTTPStatus)

	}
	return fmt.Sprintf("Error: %s", e.Error())
}

// Unwrap implements the errors.Unwrap interface for Go 1.13+ error unwrapping
func (e *Error) Unwrap() error {
	return e.cause
}

// WithDetails adds details to the error and returns the same error
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error and returns the same error
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithCause wraps another error as the cause of this error
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// ToHTTP writes the error to an HTTP response writer (for standard net/http)
func (e *Error) ToHTTP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.HTTPStatus)
	json.NewEncoder(w).Encode(e)
}

// Is implements the errors.Is interface for Go 1.13+ error comparison
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// IsCode checks if an error is an Error with a specific code
func IsCode(err error, code Code) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == code
	}
	return false
}

// IsType checks if an error is an Error with a specific type
func IsType(err error, errType Type) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == errType
	}
	return false
}

// IsHTTPStatus checks if an error is an Error with a specific HTTP status code
func IsHTTPStatus(err error, status int) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.HTTPStatus == status
	}
	return false
}

// Registry helps manage error definitions across packages
type Registry struct {
	prefix    string
	errorDefs map[Code]*Error
}

// NewRegistry creates a new Registry with a prefix
func NewRegistry(prefix string) *Registry {
	return &Registry{
		prefix:    prefix,
		errorDefs: make(map[Code]*Error),
	}
}

// Register adds a new error definition to the registry
func (r *Registry) Register(code Code, errType Type, httpStatus int, message string) Code {
	fullCode := Code(fmt.Sprintf("%s_%s", r.prefix, code))
	r.errorDefs[fullCode] = &Error{
		Code:       fullCode,
		Type:       errType,
		Message:    message,
		HTTPStatus: httpStatus,
	}
	return fullCode
}

// New creates a new instance of a registered error
func (r *Registry) New(code Code) *Error {
	if err, ok := r.errorDefs[code]; ok {
		// Create a copy of the error to avoid modifying the original definition
		return &Error{
			Code:       err.Code,
			Type:       err.Type,
			Message:    err.Message,
			HTTPStatus: err.HTTPStatus,
		}
	}
	// Return a generic internal error if the code is not found
	return &Error{
		Code:       "UNKNOWN_ERROR",
		Type:       TypeInternal,
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewWithMessage creates a new instance of a registered error with a custom message
func (r *Registry) NewWithMessage(code Code, message string) *Error {
	err := r.New(code)
	err.Message = message
	return err
}

// NewWithCause creates a new instance of a registered error with an underlying cause
func (r *Registry) NewWithCause(code Code, cause error) *Error {
	err := r.New(code)
	err.cause = cause
	return err
}

// FromResponse creates an Error from an HTTP response (for client-side use)
func FromResponse(resp *http.Response) error {
	if resp == nil {
		return &Error{
			Code:    "UNKNOWN_ERROR",
			Type:    TypeInternal,
			Message: "An unexpected error occurred",
		}
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{
			Code:    "UNKNOWN_ERROR",
			Type:    TypeInternal,
			Message: "Error reading response body",
		}
	}

	// Try to parse as Error first
	var xerr Error
	if err := json.Unmarshal(body, &xerr); err != nil {
		// If parsing fails, create a generic error with the response body as message
		return &Error{
			Code:       "EXTERNAL_ERROR",
			Type:       TypeExternal,
			Message:    string(body),
			HTTPStatus: resp.StatusCode,
		}
	}

	// Set the HTTP status from the response
	xerr.HTTPStatus = resp.StatusCode
	return &xerr
}

// FromJSON creates an Error from a JSON byte slice
func FromJSON(data []byte) (*Error, error) {
	var xerr Error
	if err := json.Unmarshal(data, &xerr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error: %w", err)
	}
	return &xerr, nil
}

// Wrap wraps a standard error with contextual information
func Wrap(err error, message string, errType Type) *Error {
	if err == nil {
		return nil
	}

	// If it's already an Error, just update the message and keep the cause chain
	var xerr *Error
	if errors.As(err, &xerr) {
		return &Error{
			Code:       xerr.Code,
			Type:       errType,
			Message:    message,
			Details:    xerr.Details,
			HTTPStatus: xerr.HTTPStatus,
			cause:      err,
		}
	}

	// Otherwise, create a new Error with the original as cause
	return &Error{
		Code:    Code(fmt.Sprintf("%s_ERROR", errType)),
		Type:    errType,
		Message: message,
		cause:   err,
	}
}

// New creates a new Error with the given message and type
func New(message string, errType Type) *Error {
	return &Error{
		Code:    Code(fmt.Sprintf("%s_ERROR", errType)),
		Type:    errType,
		Message: message,
	}
}
