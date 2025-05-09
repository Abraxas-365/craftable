package dtox

import (
	"net/http"

	"github.com/Abraxas-365/craftale/errx"
)

// ErrorRegistry holds all error definitions for the dtox package
var ErrorRegistry = errx.NewRegistry("DTOX")

// Error codes definition
var (
	// Validation errors
	ErrValidationFailed = ErrorRegistry.Register("VALIDATION_FAILED", errx.TypeValidation, http.StatusBadRequest, "Validation failed")
	ErrInvalidField     = ErrorRegistry.Register("INVALID_FIELD", errx.TypeValidation, http.StatusBadRequest, "Invalid field value")

	// Mapping errors
	ErrFieldNotFound  = ErrorRegistry.Register("FIELD_NOT_FOUND", errx.TypeBadRequest, http.StatusBadRequest, "Field not found in target type")
	ErrCannotSetField = ErrorRegistry.Register("CANNOT_SET_FIELD", errx.TypeBadRequest, http.StatusBadRequest, "Cannot set field in target type")
	ErrTypeConversion = ErrorRegistry.Register("TYPE_CONVERSION", errx.TypeBadRequest, http.StatusBadRequest, "Type conversion failed")

	// Batch operation errors
	ErrBatchConversion = ErrorRegistry.Register("BATCH_CONVERSION", errx.TypeInternal, http.StatusInternalServerError, "Batch conversion failed")
)
