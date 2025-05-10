package validatex

import (
	"errors"
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// ValidationErrorsToHTTP writes validation errors to an HTTP response
func ValidationErrorsToHTTP(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// Check if we already have an errx.Error
	var xerr *errx.Error
	if errors.As(err, &xerr) {
		xerr.ToHTTP(w)
		return
	}

	// Check if we have validation errors that can be converted
	if verrs, ok := err.(ValidationErrors); ok {
		verrs.ToErrx().ToHTTP(w)
		return
	}

	// Handle other errors
	ValidatorErrors.NewWithMessage(ErrValidationFailed, err.Error()).ToHTTP(w)
}

// ValidateRequest validates a request body and writes errors to the response if needed
// Returns true if validation passed, false if it failed
func ValidateRequest(w http.ResponseWriter, obj any) bool {
	if err := ValidateWithErrx(obj); err != nil {
		err.ToHTTP(w)
		return false
	}
	return true
}

// ValidateRequestCustom validates a request body using a custom validator
// Returns true if validation passed, false if it failed
func ValidateRequestCustom(w http.ResponseWriter, obj any, validator *CustomValidator) bool {
	if err := validator.ValidateWithErrx(obj); err != nil {
		err.ToHTTP(w)
		return false
	}
	return true
}
