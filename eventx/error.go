package eventx

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// Error registry for eventx package
var ErrorRegistry = errx.NewRegistry("EVENTX")

// Error codes
var (
	ErrEventNotFound        = ErrorRegistry.Register("EVENT_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Event not found")
	ErrInvalidEventType     = ErrorRegistry.Register("INVALID_EVENT_TYPE", errx.TypeValidation, http.StatusBadRequest, "Invalid event type")
	ErrHandlerFailed        = ErrorRegistry.Register("HANDLER_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Event handler failed")
	ErrSerializationFailed  = ErrorRegistry.Register("SERIALIZATION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Event serialization failed")
	ErrBusNotConnected      = ErrorRegistry.Register("BUS_NOT_CONNECTED", errx.TypeSystem, http.StatusServiceUnavailable, "Event bus not connected")
	ErrPublishFailed        = ErrorRegistry.Register("PUBLISH_FAILED", errx.TypeSystem, http.StatusInternalServerError, "Failed to publish event")
	ErrSubscriptionFailed   = ErrorRegistry.Register("SUBSCRIPTION_FAILED", errx.TypeSystem, http.StatusInternalServerError, "Failed to subscribe to event")
	ErrConnectionFailed     = ErrorRegistry.Register("CONNECTION_FAILED", errx.TypeSystem, http.StatusServiceUnavailable, "Failed to connect to event bus")
	ErrTimeout              = ErrorRegistry.Register("TIMEOUT", errx.TypeTimeout, http.StatusRequestTimeout, "Event operation timed out")
	ErrRateLimit            = ErrorRegistry.Register("RATE_LIMIT", errx.TypeRateLimit, http.StatusTooManyRequests, "Event rate limit exceeded")
	ErrInvalidConfiguration = ErrorRegistry.Register("INVALID_CONFIGURATION", errx.TypeValidation, http.StatusBadRequest, "Invalid event bus configuration")
)
