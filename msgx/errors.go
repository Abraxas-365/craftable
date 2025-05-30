package msgx

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

var Registry = errx.NewRegistry("MESSAGING")

// Error codes for messaging operations
var (
	ErrInvalidMessage = Registry.Register(
		"INVALID_MESSAGE",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid message format or content",
	)

	ErrProviderNotFound = Registry.Register(
		"PROVIDER_NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Messaging provider not found",
	)

	ErrSendFailed = Registry.Register(
		"SEND_FAILED",
		errx.TypeExternal,
		http.StatusBadGateway,
		"Failed to send message",
	)

	ErrWebhookVerificationFailed = Registry.Register(
		"WEBHOOK_VERIFICATION_FAILED",
		errx.TypeAuthorization,
		http.StatusUnauthorized,
		"Webhook verification failed",
	)

	ErrWebhookParseFailed = Registry.Register(
		"WEBHOOK_PARSE_FAILED",
		errx.TypeBadRequest,
		http.StatusBadRequest,
		"Failed to parse webhook payload",
	)

	ErrProviderConfigInvalid = Registry.Register(
		"PROVIDER_CONFIG_INVALID",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid provider configuration",
	)

	ErrNumberValidationFailed = Registry.Register(
		"NUMBER_VALIDATION_FAILED",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Phone number validation failed",
	)

	ErrUnsupportedMessageType = Registry.Register(
		"UNSUPPORTED_MESSAGE_TYPE",
		errx.TypeBadRequest,
		http.StatusBadRequest,
		"Unsupported message type",
	)

	ErrRateLimitExceeded = Registry.Register(
		"RATE_LIMIT_EXCEEDED",
		errx.TypeRateLimit,
		http.StatusTooManyRequests,
		"Rate limit exceeded",
	)

	ErrProviderUnavailable = Registry.Register(
		"PROVIDER_UNAVAILABLE",
		errx.TypeUnavailable,
		http.StatusServiceUnavailable,
		"Messaging provider is currently unavailable",
	)
)
