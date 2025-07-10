package msgx

import (
	"context"
	"fmt"
)

// Service manages multiple messaging providers with send and receive capabilities
type Service struct {
	senders         map[string]Sender
	receivers       map[string]Receiver
	providers       map[string]Provider // Full providers (send + receive)
	defaultSender   string
	defaultReceiver string
	eventHandler    EventHandler
	webhookServer   *WebhookServer
}

// NewService creates a new messaging service
func NewService() *Service {
	return &Service{
		senders:   make(map[string]Sender),
		receivers: make(map[string]Receiver),
		providers: make(map[string]Provider),
	}
}

// ========== Registration Methods ==========

// RegisterSender registers a send-only provider
func (s *Service) RegisterSender(name string, sender Sender, isDefault bool) *Service {
	s.senders[name] = sender
	if isDefault || s.defaultSender == "" {
		s.defaultSender = name
	}
	return s
}

// RegisterReceiver registers a receive-only provider
func (s *Service) RegisterReceiver(name string, receiver Receiver, isDefault bool) *Service {
	s.receivers[name] = receiver
	if isDefault || s.defaultReceiver == "" {
		s.defaultReceiver = name
	}
	return s
}

// RegisterProvider registers a full provider (send + receive)
func (s *Service) RegisterProvider(name string, provider Provider, isDefault bool) *Service {
	s.providers[name] = provider
	s.senders[name] = provider   // Also register as sender
	s.receivers[name] = provider // Also register as receiver

	if isDefault || (s.defaultSender == "" && s.defaultReceiver == "") {
		s.defaultSender = name
		s.defaultReceiver = name
	}
	return s
}

// ========== Sending Methods ==========

// Send sends a message using the default sender
func (s *Service) Send(ctx context.Context, message Message) (*Response, error) {
	return s.SendWithProvider(ctx, s.defaultSender, message)
}

// SendWithProvider sends a message using a specific sender
func (s *Service) SendWithProvider(ctx context.Context, providerName string, message Message) (*Response, error) {
	// Validate message
	if err := s.validateMessage(message); err != nil {
		return nil, Registry.New(ErrInvalidMessage).
			WithCause(err).
			WithDetail("message", message)
	}

	// Get sender
	sender, exists := s.senders[providerName]
	if !exists {
		return nil, Registry.New(ErrProviderNotFound).
			WithDetail("provider", providerName).
			WithDetail("available_senders", s.getSenderNames())
	}

	// Send message
	response, err := sender.Send(ctx, message)
	if err != nil {
		return nil, Registry.New(ErrSendFailed).
			WithCause(err).
			WithDetail("provider", providerName).
			WithDetail("message", message)
	}

	return response, nil
}

// SendBulk sends multiple messages using the default sender
func (s *Service) SendBulk(ctx context.Context, messages []Message) (*BulkResponse, error) {
	return s.SendBulkWithProvider(ctx, s.defaultSender, messages)
}

// SendBulkWithProvider sends multiple messages using a specific sender
func (s *Service) SendBulkWithProvider(ctx context.Context, providerName string, messages []Message) (*BulkResponse, error) {
	// Get sender
	sender, exists := s.senders[providerName]
	if !exists {
		return nil, Registry.New(ErrProviderNotFound).
			WithDetail("provider", providerName).
			WithDetail("available_senders", s.getSenderNames())
	}

	// Send bulk
	response, err := sender.SendBulk(ctx, messages)
	if err != nil {
		return nil, Registry.New(ErrSendFailed).
			WithCause(err).
			WithDetail("provider", providerName).
			WithDetail("message_count", len(messages))
	}

	return response, nil
}

// SendWithFallback tries multiple providers in order
func (s *Service) SendWithFallback(ctx context.Context, providers []string, message Message) (*Response, error) {
	var lastErr error

	for _, providerName := range providers {
		response, err := s.SendWithProvider(ctx, providerName, message)
		if err == nil {
			return response, nil
		}
		lastErr = err
	}

	return nil, Registry.New(ErrSendFailed).
		WithCause(lastErr).
		WithDetail("tried_providers", providers).
		WithDetail("message", "All providers failed")
}

// GetStatus retrieves message status using the default sender
func (s *Service) GetStatus(ctx context.Context, messageID string) (*Status, error) {
	return s.GetStatusWithProvider(ctx, s.defaultSender, messageID)
}

// GetStatusWithProvider retrieves message status using a specific sender
func (s *Service) GetStatusWithProvider(ctx context.Context, providerName string, messageID string) (*Status, error) {
	sender, exists := s.senders[providerName]
	if !exists {
		return nil, Registry.New(ErrProviderNotFound).
			WithDetail("provider", providerName).
			WithDetail("available_senders", s.getSenderNames())
	}

	status, err := sender.GetStatus(ctx, messageID)
	if err != nil {
		return nil, Registry.New(ErrSendFailed).
			WithCause(err).
			WithDetail("provider", providerName).
			WithDetail("message_id", messageID)
	}

	return status, nil
}

// SetupWebhooks configures webhooks for all registered receivers
func (s *Service) SetupWebhooks(port int, baseURL string) error {
	s.webhookServer = NewWebhookServer(port)

	for name, receiver := range s.receivers {
		// Setup webhook endpoint
		path := fmt.Sprintf("/webhook/%s", name)
		webhookURL := fmt.Sprintf("%s%s", baseURL, path)

		config := WebhookConfig{
			URL: webhookURL,
			Events: []EventType{
				EventMessageReceived,
				EventStatusUpdate,
			},
		}

		if err := receiver.SetupWebhook(config); err != nil {
			return Registry.New(ErrWebhookVerificationFailed).
				WithCause(err).
				WithDetail("provider", name).
				WithDetail("webhook_url", webhookURL)
		}

		// Register with webhook server
		s.webhookServer.RegisterProvider(path, receiver)
	}

	return nil
}

// StartWebhookServer starts the webhook server
func (s *Service) StartWebhookServer() error {
	if s.webhookServer == nil {
		return Registry.New(ErrProviderConfigInvalid).
			WithDetail("message", "Webhooks not configured")
	}
	return s.webhookServer.Start()
}

// SetEventHandler sets the event handler for incoming messages
func (s *Service) SetEventHandler(handler EventHandler) *Service {
	s.eventHandler = handler
	return s
}

// OnMessage sets a simple message handler function
func (s *Service) OnMessage(handler func(ctx context.Context, message *IncomingMessage) error) *Service {
	s.eventHandler = EventHandlerFunc(handler)
	return s
}

// ========== Helper Methods ==========

func (s *Service) validateMessage(message Message) error {
	if message.To == "" {
		return fmt.Errorf("recipient phone number is required")
	}
	if message.Type == "" {
		return fmt.Errorf("message type is required")
	}
	if message.Content.Text == nil && message.Content.Media == nil && message.Content.Template == nil {
		return fmt.Errorf("message content is required")
	}
	if message.Type == MessageTypeText && message.Content.Text == nil {
		return fmt.Errorf("text content is required for text messages")
	}
	if message.Type == MessageTypeText && message.Content.Text.Body == "" {
		return fmt.Errorf("text body is required")
	}
	return nil
}

func (s *Service) getSenderNames() []string {
	names := make([]string, 0, len(s.senders))
	for name := range s.senders {
		names = append(names, name)
	}
	return names
}

func (s *Service) getReceiverNames() []string {
	names := make([]string, 0, len(s.receivers))
	for name := range s.receivers {
		names = append(names, name)
	}
	return names
}
