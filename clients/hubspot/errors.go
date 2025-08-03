package hubspot

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// Registry is the error registry for HubSpot operations
var Registry = errx.NewRegistry("HUBSPOT")

// Error codes for HubSpot operations
var (
	// Connection errors
	ErrHubSpotConnection = Registry.Register(
		"CONNECTION_ERROR",
		errx.TypeExternal,
		http.StatusServiceUnavailable,
		"Failed to connect to HubSpot API",
	)

	ErrHubSpotTimeout = Registry.Register(
		"TIMEOUT",
		errx.TypeTimeout,
		http.StatusRequestTimeout,
		"HubSpot API request timed out",
	)

	ErrHubSpotUnavailable = Registry.Register(
		"UNAVAILABLE",
		errx.TypeUnavailable,
		http.StatusServiceUnavailable,
		"HubSpot API is unavailable",
	)

	// Authentication errors
	ErrHubSpotUnauthorized = Registry.Register(
		"UNAUTHORIZED",
		errx.TypeAuthorization,
		http.StatusUnauthorized,
		"Unauthorized access to HubSpot API",
	)

	ErrHubSpotForbidden = Registry.Register(
		"FORBIDDEN",
		errx.TypeAuthorization,
		http.StatusForbidden,
		"Forbidden access to HubSpot API",
	)

	ErrHubSpotInvalidToken = Registry.Register(
		"INVALID_TOKEN",
		errx.TypeAuthorization,
		http.StatusUnauthorized,
		"Invalid HubSpot API token",
	)

	// API errors
	ErrHubSpotAPIError = Registry.Register(
		"API_ERROR",
		errx.TypeExternal,
		http.StatusInternalServerError,
		"HubSpot API error",
	)

	ErrHubSpotRateLimit = Registry.Register(
		"RATE_LIMIT",
		errx.TypeRateLimit,
		http.StatusTooManyRequests,
		"HubSpot API rate limit exceeded",
	)

	ErrHubSpotBadRequest = Registry.Register(
		"BAD_REQUEST",
		errx.TypeBadRequest,
		http.StatusBadRequest,
		"Invalid request to HubSpot API",
	)

	ErrHubSpotNotFound = Registry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Resource not found in HubSpot",
	)

	ErrHubSpotConflict = Registry.Register(
		"CONFLICT",
		errx.TypeConflict,
		http.StatusConflict,
		"Conflict with existing HubSpot resource",
	)

	// Data errors
	ErrHubSpotInvalidData = Registry.Register(
		"INVALID_DATA",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid data provided to HubSpot API",
	)

	ErrHubSpotParsingError = Registry.Register(
		"PARSING_ERROR",
		errx.TypeInternal,
		http.StatusInternalServerError,
		"Failed to parse HubSpot API response",
	)

	// Resource-specific errors
	ErrResourceNotFound = Registry.Register(
		"RESOURCE_NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Resource not found",
	)

	ErrResourceCreateFailed = Registry.Register(
		"RESOURCE_CREATE_FAILED",
		errx.TypeExternal,
		http.StatusInternalServerError,
		"Failed to create resource",
	)

	ErrResourceUpdateFailed = Registry.Register(
		"RESOURCE_UPDATE_FAILED",
		errx.TypeExternal,
		http.StatusInternalServerError,
		"Failed to update resource",
	)

	ErrResourceDeleteFailed = Registry.Register(
		"RESOURCE_DELETE_FAILED",
		errx.TypeExternal,
		http.StatusInternalServerError,
		"Failed to delete resource",
	)
)

// NewResourceNotFoundError creates a new resource not found error
func NewResourceNotFoundError(resourceType string, resourceID interface{}) *errx.Error {
	return Registry.New(ErrResourceNotFound).
		WithDetail("resourceType", resourceType).
		WithDetail("resourceID", resourceID)
}

// NewRateLimitError creates a new rate limit error with retry information
func NewRateLimitError(retryAfter string) *errx.Error {
	err := Registry.New(ErrHubSpotRateLimit)
	if retryAfter != "" {
		err.WithDetail("retryAfter", retryAfter)
	}
	return err
}
