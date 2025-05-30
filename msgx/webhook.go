package msgx

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ========== Webhook Configuration ==========

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL           string            `json:"url" validate:"required,url"`
	Secret        string            `json:"secret,omitempty"`
	VerifyToken   string            `json:"verify_token,omitempty"`
	Events        []EventType       `json:"events"`
	Headers       map[string]string `json:"headers,omitempty"`
	RetryAttempts int               `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration     `json:"retry_delay,omitempty"`
}

// EventType represents types of events to subscribe to
type EventType string

const (
	EventMessageReceived EventType = "message.received"
	EventMessageSent     EventType = "message.sent"
	EventMessageRead     EventType = "message.read"
	EventMessageFailed   EventType = "message.failed"
	EventStatusUpdate    EventType = "status.update"
)

// ========== Webhook Server ==========

// WebhookServer manages webhook endpoints for multiple providers
type WebhookServer struct {
	handlers map[string]Receiver
	mux      *http.ServeMux
	port     int
	server   *http.Server
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int) *WebhookServer {
	return &WebhookServer{
		handlers: make(map[string]Receiver),
		mux:      http.NewServeMux(),
		port:     port,
	}
}

// RegisterProvider registers a receiver provider
func (ws *WebhookServer) RegisterProvider(path string, receiver Receiver) {
	ws.handlers[path] = receiver
	ws.mux.HandleFunc(path, ws.handleWebhook(receiver))
}

// handleWebhook creates a handler function for a specific provider
func (ws *WebhookServer) handleWebhook(receiver Receiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Verify webhook
		if err := receiver.VerifyWebhook(r); err != nil {
			Registry.New(ErrWebhookVerificationFailed).
				WithCause(err).
				WithDetail("provider", receiver.GetProviderName()).
				ToHTTP(w)
			return
		}

		// Handle webhook
		message, err := receiver.HandleWebhook(ctx, r)
		if err != nil {
			Registry.New(ErrWebhookParseFailed).
				WithCause(err).
				WithDetail("provider", receiver.GetProviderName()).
				ToHTTP(w)
			return
		}

		if message != nil {
			// Process the message (you can add custom logic here)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			// Handle verification challenges, etc.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}
}

// Start starts the webhook server
func (ws *WebhookServer) Start() error {
	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.port),
		Handler: ws.mux,
	}

	return ws.server.ListenAndServe()
}

// Stop gracefully stops the webhook server
func (ws *WebhookServer) Stop(ctx context.Context) error {
	if ws.server == nil {
		return nil
	}
	return ws.server.Shutdown(ctx)
}
