package msgx

import (
	"context"
	"net/http"
	"time"
)

// ========== Core Interfaces ==========

// Sender represents a messaging service that can send messages
type Sender interface {
	// Send sends a message and returns the message ID
	Send(ctx context.Context, message Message) (*Response, error)

	// SendBulk sends multiple messages
	SendBulk(ctx context.Context, messages []Message) (*BulkResponse, error)

	// GetStatus retrieves message delivery status
	GetStatus(ctx context.Context, messageID string) (*Status, error)

	// ValidateNumber validates if a phone number can receive messages
	ValidateNumber(ctx context.Context, phoneNumber string) (*NumberValidation, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// Receiver represents a messaging service that can receive messages
type Receiver interface {
	// SetupWebhook configures the webhook endpoint for receiving messages
	SetupWebhook(config WebhookConfig) error

	// HandleWebhook processes incoming webhook requests
	HandleWebhook(ctx context.Context, req *http.Request) (*IncomingMessage, error)

	// VerifyWebhook verifies the authenticity of incoming webhooks
	VerifyWebhook(req *http.Request) error

	// ParseIncomingMessage parses raw webhook data into structured message
	ParseIncomingMessage(data []byte) (*IncomingMessage, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// Provider represents a full-featured messaging service (send + receive)
type Provider interface {
	Sender
	Receiver
}

// SendOnlyProvider is an alias for Sender for clarity
type SendOnlyProvider = Sender

// ReceiveOnlyProvider is an alias for Receiver for clarity
type ReceiveOnlyProvider = Receiver

// ========== Message Structures ==========

// Message represents a universal message structure
type Message struct {
	To       string            `json:"to" validate:"required"`
	From     string            `json:"from,omitempty"`
	Type     MessageType       `json:"type" validate:"required"`
	Content  Content           `json:"content" validate:"required"`
	Options  *MessageOptions   `json:"options,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MessageType defines the type of message
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeDocument MessageType = "document"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeVideo    MessageType = "video"
	MessageTypeTemplate MessageType = "template"
)

// Content holds the message content based on type
type Content struct {
	Text     *TextContent     `json:"text,omitempty"`
	Media    *MediaContent    `json:"media,omitempty"`
	Template *TemplateContent `json:"template,omitempty"`
}

// TextContent for text messages
type TextContent struct {
	Body       string `json:"body" validate:"required,max=1024"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

// MediaContent for media messages
type MediaContent struct {
	URL      string `json:"url,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// TemplateContent for template messages
type TemplateContent struct {
	Name       string         `json:"name" validate:"required"`
	Language   string         `json:"language" validate:"required"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// MessageOptions for additional message settings
type MessageOptions struct {
	Priority    Priority  `json:"priority,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	TTL         int       `json:"ttl,omitempty"` // Time to live in seconds
	Webhook     string    `json:"webhook,omitempty"`
}

// Priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// ========== Response Structures ==========

// Response represents the response from sending a message
type Response struct {
	MessageID       string           `json:"message_id"`
	Provider        string           `json:"provider"`
	To              string           `json:"to"`
	Status          MessageStatus    `json:"status"`
	Cost            *Cost            `json:"cost,omitempty"`
	Timestamp       time.Time        `json:"timestamp"`
	ProviderData    map[string]any   `json:"provider_data,omitempty"`
	ResolvedContent *ResolvedContent `json:"resolved_content,omitempty"` // Add this
}

// ResolvedContent holds the resolved template information
type ResolvedContent struct {
	TemplateName    string         `json:"template_name"`
	Language        string         `json:"language"`
	OriginalBody    string         `json:"original_body,omitempty"`
	ResolvedBody    string         `json:"resolved_body"`
	ResolvedMessage string         `json:"resolved_message"`
	Parameters      map[string]any `json:"parameters,omitempty"`
	Header          string         `json:"header,omitempty"`
	Footer          string         `json:"footer,omitempty"`
	ParameterCount  int            `json:"parameter_count"`
}

// BulkResponse for bulk operations
type BulkResponse struct {
	TotalSent   int           `json:"total_sent"`
	TotalFailed int           `json:"total_failed"`
	Responses   []Response    `json:"responses"`
	FailedItems []BulkFailure `json:"failed_items,omitempty"`
}

// BulkFailure represents a failed item in bulk operation
type BulkFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// Status represents message delivery status
type Status struct {
	MessageID string        `json:"message_id"`
	Status    MessageStatus `json:"status"`
	UpdatedAt time.Time     `json:"updated_at"`
	ErrorCode string        `json:"error_code,omitempty"`
	ErrorMsg  string        `json:"error_message,omitempty"`
}

// MessageStatus represents the delivery status
type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// NumberValidation represents number validation result
type NumberValidation struct {
	PhoneNumber string `json:"phone_number"`
	IsValid     bool   `json:"is_valid"`
	Carrier     string `json:"carrier,omitempty"`
	Country     string `json:"country,omitempty"`
	LineType    string `json:"line_type,omitempty"`
}

// Cost represents messaging cost
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Unit     string  `json:"unit"` // "message", "segment", etc.
}

// ========== Incoming Message Structures ==========

// IncomingMessage represents a message received via webhook
type IncomingMessage struct {
	ID        string          `json:"id"`
	Provider  string          `json:"provider"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Type      MessageType     `json:"type"`
	Content   IncomingContent `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
	Status    MessageStatus   `json:"status,omitempty"`
	Context   *MessageContext `json:"context,omitempty"`
	RawData   map[string]any  `json:"raw_data,omitempty"`
}

// IncomingContent represents the content of an incoming message
type IncomingContent struct {
	Text     *IncomingTextContent  `json:"text,omitempty"`
	Media    *IncomingMediaContent `json:"media,omitempty"`
	Location *LocationContent      `json:"location,omitempty"`
	Contact  *ContactContent       `json:"contact,omitempty"`
}

// IncomingTextContent for incoming text messages
type IncomingTextContent struct {
	Body string `json:"body"`
}

// IncomingMediaContent for incoming media messages
type IncomingMediaContent struct {
	URL      string `json:"url,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// LocationContent for location messages
type LocationContent struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ContactContent for contact messages
type ContactContent struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email,omitempty"`
}

// MessageContext provides context about the conversation
type MessageContext struct {
	ConversationID string `json:"conversation_id,omitempty"`
	ReplyToID      string `json:"reply_to_id,omitempty"`
	IsForwarded    bool   `json:"is_forwarded,omitempty"`
	ForwardedFrom  string `json:"forwarded_from,omitempty"`
}

// ========== Event Handling ==========

// EventHandler handles incoming message events
type EventHandler interface {
	// HandleMessage processes incoming messages
	HandleMessage(ctx context.Context, message *IncomingMessage) error

	// HandleStatusUpdate processes message status updates
	HandleStatusUpdate(ctx context.Context, status *Status) error

	// HandleError processes webhook errors
	HandleError(ctx context.Context, err error, rawData []byte) error
}

// EventHandlerFunc is a function adapter for EventHandler
type EventHandlerFunc func(ctx context.Context, message *IncomingMessage) error

func (f EventHandlerFunc) HandleMessage(ctx context.Context, message *IncomingMessage) error {
	return f(ctx, message)
}

func (f EventHandlerFunc) HandleStatusUpdate(ctx context.Context, status *Status) error {
	return nil // Default implementation
}

func (f EventHandlerFunc) HandleError(ctx context.Context, err error, rawData []byte) error {
	return nil // Default implementation
}
