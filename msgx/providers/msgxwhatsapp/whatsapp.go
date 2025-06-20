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
	// Handle verification challenge (GET request)
	if req.Method == "GET" {
		return w.handleVerificationChallenge(req)
	}

	// Handle POST requests (actual messages)
	if req.Method != "POST" {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid HTTP method").
			WithDetail("method", req.Method)
	}

	// Read body once and store it
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	// Log the raw webhook payload for debugging
	logx.Debug("WhatsApp webhook raw payload (%d bytes): %s", len(body), string(body))

	// Verify webhook signature using the exact body bytes we just read
	if err := w.verifyWebhookSignature(req, body); err != nil {
		return nil, err
	}

	// Parse the webhook body using the same bytes
	return w.ParseIncomingMessage(body)
}

// handleVerificationChallenge handles WhatsApp verification challenges
func (w *WhatsAppProvider) handleVerificationChallenge(req *http.Request) (*msgx.IncomingMessage, error) {
	query := req.URL.Query()
	mode := query.Get(modeParam)
	token := query.Get(verifyTokenParam)
	challenge := query.Get(challengeParam)

	logx.Debug("WhatsApp verification challenge - Mode: %s, Token provided: %t, Challenge: %s",
		mode, token != "", challenge)

	if mode == "subscribe" && token == w.config.VerifyToken {
		// Return a special message indicating verification success
		// The actual HTTP response should be handled at the handler level
		return &msgx.IncomingMessage{
			Provider: whatsappProvider,
			Type:     "verification_challenge",
			RawData:  map[string]any{"challenge": challenge},
		}, nil
	}

	return nil, msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
		WithDetail("provider", whatsappProvider).
		WithDetail("reason", "Invalid verification token").
		WithDetail("mode", mode).
		WithDetail("token_provided", token != "").
		WithDetail("expected_token", w.config.VerifyToken).
		WithDetail("challenge", challenge)
}

// VerifyWebhook verifies the webhook signature (kept for compatibility)
func (w *WhatsAppProvider) VerifyWebhook(req *http.Request) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	// Reset body
	req.Body = io.NopCloser(bytes.NewReader(body))

	return w.verifyWebhookSignature(req, body)
}

// verifyWebhookSignature verifies the webhook signature with body already read
func (w *WhatsAppProvider) verifyWebhookSignature(req *http.Request, body []byte) error {
	if w.config.WebhookSecret == "" {
		logx.Debug("WhatsApp webhook signature verification skipped - no secret configured")
		return nil // Skip verification if no secret configured
	}

	signature := req.Header.Get(signatureHeader)
	if signature == "" {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Missing signature header")
	}

	// Log for debugging
	logx.Debug("WhatsApp signature verification:")
	logx.Debug("  Webhook secret: '%s'", w.config.WebhookSecret)
	logx.Debug("  Body length: %d bytes", len(body))
	logx.Debug("  Raw body: %s", string(body))
	logx.Debug("  Received signature header: %s", signature)

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")
	logx.Debug("  Signature without prefix: %s", signature)

	// Calculate expected signature using the EXACT body bytes
	mac := hmac.New(sha256.New, []byte(w.config.WebhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	logx.Debug("  Expected signature: %s", expectedSignature)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		logx.Error("WhatsApp webhook signature verification failed - Expected: %s, Got: %s",
			expectedSignature, signature)

		// Additional debugging - let's try with different approaches
		logx.Debug("Debug signature verification attempts:")

		// Try with string conversion
		mac2 := hmac.New(sha256.New, []byte(w.config.WebhookSecret))
		mac2.Write([]byte(string(body)))
		expectedSignature2 := hex.EncodeToString(mac2.Sum(nil))
		logx.Debug("  With string conversion: %s", expectedSignature2)

		// Check if there are any character encoding issues
		logx.Debug("  Body as hex: %x", body)
		logx.Debug("  Secret as hex: %x", w.config.WebhookSecret)

		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid signature").
			WithDetail("expected", expectedSignature).
			WithDetail("received", signature).
			WithDetail("body_length", len(body)).
			WithDetail("secret_length", len(w.config.WebhookSecret))
	}

	logx.Debug("WhatsApp webhook signature verification successful")
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

	// Pretty print for debugging
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err == nil {
		logx.Debug("WhatsApp webhook parsed JSON:\n%s", prettyJSON.String())
	}

	// WhatsApp webhooks can contain multiple entries
	for _, entry := range webhook.Entry {
		logx.Debug("Processing entry ID: %s with %d changes", entry.ID, len(entry.Changes))

		for _, change := range entry.Changes {
			logx.Debug("Processing change field: %s", change.Field)

			// Handle incoming messages
			if change.Value.Messages != nil && len(change.Value.Messages) > 0 {
				for _, msg := range change.Value.Messages {
					logx.Debug("Processing message ID: %s, Type: %s, From: %s",
						msg.ID, msg.Type, msg.From)

					incomingMsg := w.convertToIncomingMessage(msg, change.Value.Metadata)
					logx.Info("WhatsApp message received - ID: %s, From: %s, Type: %s",
						incomingMsg.ID, incomingMsg.From, incomingMsg.Type)

					return incomingMsg, nil
				}
			}

			// Handle status updates
			if change.Value.Statuses != nil && len(change.Value.Statuses) > 0 {
				logx.Debug("Received %d status updates, ignoring for incoming message processing",
					len(change.Value.Statuses))
				// Return nil for status updates as they're handled separately
				return nil, nil
			}
		}
	}

	logx.Debug("No messages found in WhatsApp webhook")
	return nil, nil
}

// resolveMediaURL resolves media URL from media ID
func (w *WhatsAppProvider) resolveMediaURL(ctx context.Context, mediaID string) (string, error) {
	url := fmt.Sprintf("%s/%s", w.baseURL, mediaID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to resolve media URL: %d", resp.StatusCode)
	}

	var mediaResp struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
		return "", err
	}

	return mediaResp.URL, nil
}

// ========== Helper Methods ==========

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

	// Add debug logging
	logx.Debug("Sending WhatsApp message to URL: %s", url)
	logx.Debug("WhatsApp message payload: %s", string(jsonData))

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

	// Log response for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	logx.Debug("WhatsApp API response status: %d, body: %s", resp.StatusCode, string(bodyBytes))

	// Reset body for JSON decoding
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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

	// WhatsApp requires numbers WITHOUT the + prefix in the API
	if strings.HasPrefix(cleaned, "+") {
		cleaned = cleaned[1:]
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
		RawData:   map[string]any{"whatsapp_message": msg, "metadata": metadata},
	}

	logx.Debug("Processing WhatsApp message type: %s", msg.Type)

	// Handle different message types
	switch msg.Type {
	case "text":
		if msg.Text.Body != "" {
			incomingMsg.Content.Text = &msgx.IncomingTextContent{
				Body: msg.Text.Body,
			}
			logx.Debug("Text message body: %s", msg.Text.Body)
		}

	case "image":
		mediaURL := msg.Image.Link
		if mediaURL == "" && msg.Image.ID != "" {
			logx.Debug("Image URL not provided, media ID: %s", msg.Image.ID)
			// Note: You might need to resolve the media URL using the ID
			// For now, we'll use the ID as a placeholder
			mediaURL = fmt.Sprintf("whatsapp://media/%s", msg.Image.ID)
		}

		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      mediaURL,
			Caption:  msg.Image.Caption,
			MimeType: msg.Image.MimeType,
		}
		logx.Debug("Image message - URL: %s, Caption: %s", mediaURL, msg.Image.Caption)

	case "document":
		mediaURL := msg.Document.Link
		if mediaURL == "" && msg.Document.ID != "" {
			mediaURL = fmt.Sprintf("whatsapp://media/%s", msg.Document.ID)
		}

		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      mediaURL,
			Caption:  msg.Document.Caption,
			Filename: msg.Document.Filename,
			MimeType: msg.Document.MimeType,
		}

	case "audio":
		mediaURL := msg.Audio.Link
		if mediaURL == "" && msg.Audio.ID != "" {
			mediaURL = fmt.Sprintf("whatsapp://media/%s", msg.Audio.ID)
		}

		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      mediaURL,
			Caption:  msg.Audio.Caption,
			MimeType: msg.Audio.MimeType,
		}

	case "video":
		mediaURL := msg.Video.Link
		if mediaURL == "" && msg.Video.ID != "" {
			mediaURL = fmt.Sprintf("whatsapp://media/%s", msg.Video.ID)
		}

		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			URL:      mediaURL,
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

	default:
		logx.Warn("Unknown WhatsApp message type: %s", msg.Type)
		// Set as text type for unknown messages
		incomingMsg.Type = msgx.MessageTypeText
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: fmt.Sprintf("Unsupported message type: %s", msg.Type),
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

// ========== HTTP Handler ==========

// WhatsAppWebhookHandler creates an HTTP handler for WhatsApp webhooks
func WhatsAppWebhookHandler(provider *WhatsAppProvider, messageProcessor func(*msgx.IncomingMessage) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Add request logging
		logx.Debug("WhatsApp webhook received - Method: %s, URL: %s", r.Method, r.URL.String())

		if r.Method == "GET" {
			// Handle verification challenge
			query := r.URL.Query()
			mode := query.Get("hub.mode")
			token := query.Get("hub.verify_token")
			challenge := query.Get("hub.challenge")

			logx.Debug("Verification challenge - Mode: %s, Token provided: %t, Challenge: %s",
				mode, token != "", challenge)

			if mode == "subscribe" && token == provider.config.VerifyToken {
				logx.Info("WhatsApp webhook verification successful")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}

			logx.Error("WhatsApp webhook verification failed - invalid token")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.Method == "POST" {
			// Handle incoming messages
			incomingMsg, err := provider.HandleWebhook(ctx, r)
			if err != nil {
				logx.Error("WhatsApp webhook processing failed: %v", err)

				// Still return 200 to WhatsApp to avoid retries for permanent failures
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ERROR"))
				return
			}

			if incomingMsg != nil {
				// Check if this is a verification challenge response
				if incomingMsg.Type == "verification_challenge" {
					if challenge, ok := incomingMsg.RawData["challenge"].(string); ok {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(challenge))
						return
					}
				}

				logx.Info("WhatsApp message received from %s: %s", incomingMsg.From, incomingMsg.Type)

				// Process the incoming message
				if messageProcessor != nil {
					if err := messageProcessor(incomingMsg); err != nil {
						logx.Error("Failed to process WhatsApp message: %v", err)
					}
				}

				// Log message details
				if incomingMsg.Content.Text != nil {
					logx.Debug("Text message content: %s", incomingMsg.Content.Text.Body)
				}
			} else {
				logx.Debug("WhatsApp webhook processed but no message returned (likely status update)")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// DebugWhatsAppWebhook creates a debug handler to inspect webhook payloads
func DebugWhatsAppWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			query := r.URL.Query()
			challenge := query.Get("hub.challenge")
			logx.Debug("Debug: WhatsApp verification challenge: %s", challenge)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}

		if r.Method == "POST" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			logx.Debug("Raw WhatsApp webhook payload: %s", string(body))
			logx.Debug("WhatsApp webhook headers: %+v", r.Header)

			// Pretty print JSON
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
				logx.Debug("Pretty WhatsApp webhook JSON:\n%s", prettyJSON.String())
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DEBUG_OK"))
		}
	}
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

type whatsappContact struct {
	Profile whatsappProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type whatsappProfile struct {
	Name string `json:"name"`
}

type whatsappIncomingMessage struct {
	ID        string                    `json:"id"`
	From      string                    `json:"from"`
	Timestamp string                    `json:"timestamp"`
	Type      string                    `json:"type"`
	Context   whatsappMessageContext    `json:"context"`
	Text      whatsappIncomingText      `json:"text"`
	Image     whatsappIncomingMedia     `json:"image"`
	Document  whatsappIncomingDocument  `json:"document"`
	Audio     whatsappIncomingMedia     `json:"audio"`
	Video     whatsappIncomingMedia     `json:"video"`
	Location  whatsappIncomingLocation  `json:"location"`
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
	Timestamp   string `json:"timestamp"`
	RecipientID string `json:"recipient_id"`
}

// VerificationChallengeResponse represents a webhook verification challenge
type VerificationChallengeResponse struct {
	Challenge string
}

func (v *VerificationChallengeResponse) Error() string {
	return "webhook_verification_challenge"
}
