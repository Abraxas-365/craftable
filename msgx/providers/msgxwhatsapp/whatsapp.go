package msgxwhatsapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/logx"
	"github.com/Abraxas-365/craftable/msgx"
)

const (
	whatsappAPIURL   = "https://graph.facebook.com/"
	whatsappProvider = "whatsapp"
	signatureHeader  = "X-Hub-Signature-256"
	challengeParam   = "hub.challenge"
	verifyTokenParam = "hub.verify_token"
	modeParam        = "hub.mode"
)

// WhatsAppConfig holds WhatsApp Business API configuration
type WhatsAppConfig struct {
	PhoneNumberID string `json:"phone_number_id" validate:"required"`
	AccessToken   string `json:"access_token" validate:"required"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	VerifyToken   string `json:"verify_token,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
	HTTPTimeout   int    `json:"http_timeout,omitempty"`
}

// WhatsAppProvider implements the msgx.Provider interface
type WhatsAppProvider struct {
	config     WhatsAppConfig
	httpClient *http.Client
	baseURL    string
}

// NewWhatsAppProvider creates a new WhatsApp provider
func NewWhatsAppProvider(config WhatsAppConfig) *WhatsAppProvider {
	if config.APIVersion == "" {
		config.APIVersion = "v18.0"
	}
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30
	}

	return &WhatsAppProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		},
		baseURL: fmt.Sprintf("%s/%s", whatsappAPIURL, config.APIVersion),
	}
}

// ========== Sender Interface Implementation ==========

// Send sends a message via WhatsApp Business API
func (w *WhatsAppProvider) Send(ctx context.Context, message msgx.Message) (*msgx.Response, error) {
	// Convert to WhatsApp API format
	whatsappMsg, err := w.convertToWhatsAppMessage(message)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrInvalidMessage).
			WithCause(err).
			WithDetail("message", message).
			WithDetail("provider", whatsappProvider)
	}

	// Send via API
	response, err := w.sendMessage(ctx, whatsappMsg)
	if err != nil {
		return nil, err
	}

	return &msgx.Response{
		MessageID:    response.Messages[0].ID,
		Provider:     whatsappProvider,
		To:           message.To,
		Status:       msgx.StatusSent,
		Timestamp:    time.Now(),
		ProviderData: map[string]any{"whatsapp_message_id": response.Messages[0].ID},
	}, nil
}

// SendBulk sends multiple messages
func (w *WhatsAppProvider) SendBulk(ctx context.Context, messages []msgx.Message) (*msgx.BulkResponse, error) {
	responses := make([]msgx.Response, 0, len(messages))
	failures := make([]msgx.BulkFailure, 0)
	totalSent := 0

	for i, message := range messages {
		response, err := w.Send(ctx, message)
		if err != nil {
			failures = append(failures, msgx.BulkFailure{
				Index:   i,
				Message: message.To,
				Error:   err.Error(),
			})
			continue
		}
		responses = append(responses, *response)
		totalSent++
	}

	return &msgx.BulkResponse{
		TotalSent:   totalSent,
		TotalFailed: len(failures),
		Responses:   responses,
		FailedItems: failures,
	}, nil
}

// GetStatus retrieves message status
func (w *WhatsAppProvider) GetStatus(ctx context.Context, messageID string) (*msgx.Status, error) {
	url := fmt.Sprintf("%s/%s", w.baseURL, messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("operation", "get_status").
			WithDetail("message_id", messageID).
			WithDetail("provider", whatsappProvider)
	}

	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("message_id", messageID).
			WithDetail("operation", "http_request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, w.handleAPIError(resp)
	}

	var statusResp whatsappStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("operation", "decode_status_response").
			WithDetail("provider", whatsappProvider)
	}

	return &msgx.Status{
		MessageID: messageID,
		Status:    w.convertWhatsAppStatus(statusResp.Status),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateNumber validates a phone number
func (w *WhatsAppProvider) ValidateNumber(ctx context.Context, phoneNumber string) (*msgx.NumberValidation, error) {
	// WhatsApp doesn't have a direct validation API, but we can check format
	cleaned := w.cleanPhoneNumber(phoneNumber)
	isValid := len(cleaned) >= 10 && len(cleaned) <= 15

	if !isValid {
		return nil, msgx.Registry.New(msgx.ErrNumberValidationFailed).
			WithDetail("phone_number", phoneNumber).
			WithDetail("cleaned", cleaned).
			WithDetail("reason", "Invalid length").
			WithDetail("provider", whatsappProvider)
	}

	return &msgx.NumberValidation{
		PhoneNumber: cleaned,
		IsValid:     isValid,
		LineType:    "mobile", // WhatsApp is mobile-only
	}, nil
}

// GetProviderName returns the provider name
func (w *WhatsAppProvider) GetProviderName() string {
	return whatsappProvider
}

// ========== Receiver Interface Implementation ==========

// SetupWebhook configures the webhook endpoint
func (w *WhatsAppProvider) SetupWebhook(config msgx.WebhookConfig) error {
	// Store webhook config for verification
	if config.Secret != "" {
		w.config.WebhookSecret = config.Secret
	}

	// WhatsApp webhook setup is typically done through the Facebook Developer Console
	// This method can be used to validate the configuration

	return nil
}

// HandleWebhook processes incoming webhook requests
func (w *WhatsAppProvider) HandleWebhook(ctx context.Context, req *http.Request) (*msgx.IncomingMessage, error) {
	// Handle verification challenge
	if req.Method == "GET" {
		query := req.URL.Query()
		mode := query.Get(modeParam)
		token := query.Get(verifyTokenParam)
		challenge := query.Get(challengeParam)

		if mode == "subscribe" && token == w.config.VerifyToken {
			// For verification challenges, we need to return the challenge
			// This should be handled at the HTTP handler level
			// We'll return a special error that the webhook server can detect
			return nil, &VerificationChallengeResponse{Challenge: challenge}
		}

		return nil, msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid verification token").
			WithDetail("mode", mode).
			WithDetail("token_provided", token != "").
			WithDetail("challenge", challenge)
	}

	// Verify webhook signature for POST requests
	if err := w.VerifyWebhook(req); err != nil {
		return nil, err
	}

	// Parse webhook body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	return w.ParseIncomingMessage(body)
}

// VerifyWebhook verifies the webhook signature
func (w *WhatsAppProvider) VerifyWebhook(req *http.Request) error {
	if w.config.WebhookSecret == "" {
		return nil // Skip verification if no secret configured
	}

	signature := req.Header.Get(signatureHeader)
	if signature == "" {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Missing signature header")
	}

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	// Reset body for subsequent reads
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(w.config.WebhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid signature")
	}

	return nil
}

// ParseIncomingMessage parses webhook data into structured message
func (w *WhatsAppProvider) ParseIncomingMessage(data []byte) (*msgx.IncomingMessage, error) {
	var webhook whatsappWebhook
	if err := json.Unmarshal(data, &webhook); err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "unmarshal_webhook")
	}

	// WhatsApp webhooks can contain multiple entries
	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if change.Value.Messages != nil {
				for _, msg := range change.Value.Messages {
					return w.convertToIncomingMessage(msg, change.Value.Metadata), nil
				}
			}

			// Handle status updates
			if change.Value.Statuses != nil {
				// Return nil for status updates as they're handled separately
				return nil, nil
			}
		}
	}

	return nil, nil
}

// ========== Helper Methods ==========

func (w *WhatsAppProvider) handleVerificationChallenge(req *http.Request) (*msgx.IncomingMessage, error) {
	query := req.URL.Query()
	mode := query.Get(modeParam)
	token := query.Get(verifyTokenParam)
	challenge := query.Get(challengeParam)

	if mode == "subscribe" && token == w.config.VerifyToken {
		// For WhatsApp verification, we need to echo back the challenge
		// Since this function returns IncomingMessage, we'll handle the challenge differently
		// The actual HTTP response should be handled at the webhook server level
		return nil, nil
	}

	return nil, msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
		WithDetail("provider", whatsappProvider).
		WithDetail("reason", "Invalid verification token").
		WithDetail("mode", mode).
		WithDetail("token_provided", token != "").
		WithDetail("challenge", challenge)
}

func (w *WhatsAppProvider) convertToWhatsAppMessage(msg msgx.Message) (*whatsappMessage, error) {
	whatsappMsg := &whatsappMessage{
		MessagingProduct: "whatsapp",
		To:               w.cleanPhoneNumber(msg.To),
		Type:             string(msg.Type),
	}

	switch msg.Type {
	case msgx.MessageTypeText:
		if msg.Content.Text == nil {
			return nil, fmt.Errorf("text content is required for text messages")
		}
		whatsappMsg.Text = &whatsappTextContent{
			Body:       msg.Content.Text.Body,
			PreviewURL: msg.Content.Text.PreviewURL,
		}

	case msgx.MessageTypeImage, msgx.MessageTypeDocument, msgx.MessageTypeAudio, msgx.MessageTypeVideo:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for media messages")
		}

		mediaType := strings.ToLower(string(msg.Type))
		whatsappMsg.Type = mediaType

		mediaContent := map[string]any{
			"link": msg.Content.Media.URL,
		}
		if msg.Content.Media.Caption != "" {
			mediaContent["caption"] = msg.Content.Media.Caption
		}
		if msg.Content.Media.Filename != "" {
			mediaContent["filename"] = msg.Content.Media.Filename
		}

		switch mediaType {
		case "image":
			whatsappMsg.Image = mediaContent
		case "document":
			whatsappMsg.Document = mediaContent
		case "audio":
			whatsappMsg.Audio = mediaContent
		case "video":
			whatsappMsg.Video = mediaContent
		}

	case msgx.MessageTypeTemplate:
		if msg.Content.Template == nil {
			return nil, fmt.Errorf("template content is required for template messages")
		}
		whatsappMsg.Template = &whatsappTemplateContent{
			Name:     msg.Content.Template.Name,
			Language: whatsappLanguage{Code: msg.Content.Template.Language},
		}

		// Convert parameters
		if len(msg.Content.Template.Parameters) > 0 {
			components := make([]whatsappComponent, 0)
			parameters := make([]whatsappParameter, 0)

			for _, value := range msg.Content.Template.Parameters {
				parameters = append(parameters, whatsappParameter{
					Type: "text",
					Text: fmt.Sprintf("%v", value),
				})
			}

			if len(parameters) > 0 {
				components = append(components, whatsappComponent{
					Type:       "body",
					Parameters: parameters,
				})
			}

			whatsappMsg.Template.Components = components
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	return whatsappMsg, nil
}

func (w *WhatsAppProvider) sendMessage(ctx context.Context, message *whatsappMessage) (*whatsappSendResponse, error) {
	url := fmt.Sprintf("%s/%s/messages", w.baseURL, w.config.PhoneNumberID)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "marshal_message")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "create_request")
	}

	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "http_request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, w.handleAPIError(resp)
	}

	var sendResp whatsappSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "decode_response")
	}

	return &sendResp, nil
}

func (w *WhatsAppProvider) handleAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errorResp whatsappErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Code != 0 {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return msgx.Registry.New(msgx.ErrRateLimitExceeded).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp.Error).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusServiceUnavailable:
			return msgx.Registry.New(msgx.ErrProviderUnavailable).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp.Error).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusUnauthorized:
			return msgx.Registry.New(msgx.ErrProviderConfigInvalid).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp.Error).
				WithDetail("reason", "Invalid access token")
		case http.StatusBadRequest:
			return msgx.Registry.New(msgx.ErrInvalidMessage).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp.Error).
				WithDetail("http_status", resp.StatusCode)
		default:
			return msgx.Registry.New(msgx.ErrSendFailed).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp.Error).
				WithDetail("http_status", resp.StatusCode)
		}
	}

	return msgx.Registry.New(msgx.ErrSendFailed).
		WithDetail("provider", whatsappProvider).
		WithDetail("http_status", resp.StatusCode).
		WithDetail("response_body", string(body))
}

func (w *WhatsAppProvider) cleanPhoneNumber(phoneNumber string) string {
	// Remove all non-digit characters except '+'
	cleaned := ""
	for _, char := range phoneNumber {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		} else if char == '+' && len(cleaned) == 0 {
			cleaned += string(char)
		}
	}
	return cleaned
}

func (w *WhatsAppProvider) convertWhatsAppStatus(status string) msgx.MessageStatus {
	switch status {
	case "sent":
		return msgx.StatusSent
	case "delivered":
		return msgx.StatusDelivered
	case "read":
		return msgx.StatusRead
	case "failed":
		return msgx.StatusFailed
	default:
		return msgx.StatusPending
	}
}

// Updated convertToIncomingMessage function to handle string timestamps
func (w *WhatsAppProvider) convertToIncomingMessage(msg whatsappIncomingMessage, metadata whatsappMetadata) *msgx.IncomingMessage {
	// Parse timestamp from string to int64, then to time.Time
	var timestamp time.Time
	if msg.Timestamp != "" {
		if timestampInt, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
			timestamp = time.Unix(timestampInt, 0)
		} else {
			// Fallback to current time if parsing fails
			timestamp = time.Now()
			logx.Warn("Failed to parse WhatsApp timestamp '%s': %v", msg.Timestamp, err)
		}
	} else {
		timestamp = time.Now()
	}

	incomingMsg := &msgx.IncomingMessage{
		ID:        msg.ID,
		Provider:  whatsappProvider,
		From:      msg.From,
		To:        metadata.PhoneNumberID,
		Type:      msgx.MessageType(msg.Type),
		Timestamp: timestamp,
		Content:   msgx.IncomingContent{},
	}

	// Handle different message types
	switch msg.Type {
	case "text":
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: msg.Text.Body,
		}

	case "image":
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      msg.Image.Link,
			Caption:  msg.Image.Caption,
			MimeType: msg.Image.MimeType,
		}

	case "document":
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      msg.Document.Link,
			Caption:  msg.Document.Caption,
			Filename: msg.Document.Filename,
			MimeType: msg.Document.MimeType,
		}

	case "audio":
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      msg.Audio.Link,
			Caption:  msg.Audio.Caption,
			MimeType: msg.Audio.MimeType,
		}

	case "video":
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      msg.Video.Link,
			Caption:  msg.Video.Caption,
			MimeType: msg.Video.MimeType,
		}

	case "location":
		incomingMsg.Content.Location = &msgx.LocationContent{
			Latitude:  msg.Location.Latitude,
			Longitude: msg.Location.Longitude,
			Name:      msg.Location.Name,
			Address:   msg.Location.Address,
		}

	case "contacts":
		if len(msg.Contacts) > 0 {
			contact := msg.Contacts[0] // Take first contact
			incomingMsg.Content.Contact = &msgx.ContactContent{
				Name: contact.Name.FormattedName,
			}
			if len(contact.Phones) > 0 {
				incomingMsg.Content.Contact.PhoneNumber = contact.Phones[0].Phone
			}
		}
	}

	// Add context if available
	if msg.Context.From != "" {
		incomingMsg.Context = &msgx.MessageContext{
			ReplyToID: msg.Context.ID,
		}
	}

	return incomingMsg
}

// ========== WhatsApp API Structures ==========

type whatsappMessage struct {
	MessagingProduct string                   `json:"messaging_product"`
	To               string                   `json:"to"`
	Type             string                   `json:"type"`
	Text             *whatsappTextContent     `json:"text,omitempty"`
	Image            map[string]any           `json:"image,omitempty"`
	Document         map[string]any           `json:"document,omitempty"`
	Audio            map[string]any           `json:"audio,omitempty"`
	Video            map[string]any           `json:"video,omitempty"`
	Template         *whatsappTemplateContent `json:"template,omitempty"`
}

type whatsappTextContent struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type whatsappTemplateContent struct {
	Name       string              `json:"name"`
	Language   whatsappLanguage    `json:"language"`
	Components []whatsappComponent `json:"components,omitempty"`
}

type whatsappLanguage struct {
	Code string `json:"code"`
}

type whatsappComponent struct {
	Type       string              `json:"type"`
	Parameters []whatsappParameter `json:"parameters,omitempty"`
}

type whatsappParameter struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type whatsappSendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type whatsappStatusResponse struct {
	Status string `json:"status"`
}

type whatsappErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

type whatsappWebhook struct {
	Object string          `json:"object"`
	Entry  []whatsappEntry `json:"entry"`
}

type whatsappEntry struct {
	ID      string           `json:"id"`
	Changes []whatsappChange `json:"changes"`
}

type whatsappChange struct {
	Value whatsappValue `json:"value"`
	Field string        `json:"field"`
}

type whatsappValue struct {
	MessagingProduct string                    `json:"messaging_product"`
	Metadata         whatsappMetadata          `json:"metadata"`
	Contacts         []whatsappContact         `json:"contacts,omitempty"`
	Messages         []whatsappIncomingMessage `json:"messages,omitempty"`
	Statuses         []whatsappStatus          `json:"statuses,omitempty"`
}

type whatsappMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// Fixed: JSON tag should be lowercase "profile"
type whatsappContact struct {
	Profile whatsappProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

// Fixed: JSON tag should be lowercase "name"
type whatsappProfile struct {
	Name string `json:"name"`
}

type whatsappIncomingMessage struct {
	ID        string                    `json:"id"`
	From      string                    `json:"from"`
	Timestamp string                    `json:"timestamp"` // Changed from int64 to string
	Type      string                    `json:"type"`
	Context   whatsappMessageContext    `json:"context,omitempty"`
	Text      whatsappIncomingText      `json:"text,omitempty"` // FIXED: Complete JSON tag
	Image     whatsappIncomingMedia     `json:"image,omitempty"`
	Document  whatsappIncomingDocument  `json:"document,omitempty"`
	Audio     whatsappIncomingMedia     `json:"audio,omitempty"`
	Video     whatsappIncomingMedia     `json:"video,omitempty"`
	Location  whatsappIncomingLocation  `json:"location,omitempty"`
	Contacts  []whatsappIncomingContact `json:"contacts,omitempty"`
}

type whatsappMessageContext struct {
	From      string `json:"from,omitempty"`
	ID        string `json:"id,omitempty"`
	Referred  bool   `json:"referred,omitempty"`
	Forwarded bool   `json:"forwarded,omitempty"`
}

type whatsappIncomingText struct {
	Body string `json:"body"`
}

type whatsappIncomingMedia struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
}

type whatsappIncomingDocument struct {
	whatsappIncomingMedia
	Filename string `json:"filename,omitempty"`
}

type whatsappIncomingLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type whatsappIncomingContact struct {
	Name   whatsappContactName    `json:"name"`
	Phones []whatsappContactPhone `json:"phones,omitempty"`
}

type whatsappContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
}

type whatsappContactPhone struct {
	Phone string `json:"phone"`
	Type  string `json:"type,omitempty"`
}

type whatsappStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"` // Changed from int64 to string
	RecipientID string `json:"recipient_id"`
}

// VerificationChallengeResponse represents a webhook verification challenge
type VerificationChallengeResponse struct {
	Challenge string
}

func (v *VerificationChallengeResponse) Error() string {
	return "webhook_verification_challenge"
}
