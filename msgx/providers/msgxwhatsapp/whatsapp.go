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
	"regexp"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/msgx"
)

const (
	whatsappAPIURL          = "https://graph.facebook.com"
	whatsappProvider        = "whatsapp"
	whatsappSignatureHeader = "X-Hub-Signature-256"
	whatsappAPIVersion      = "v23.0" // Updated to latest version
)

// WhatsAppConfig holds WhatsApp Business API configuration
type WhatsAppConfig struct {
	AccessToken   string `json:"access_token" validate:"required"`
	PhoneNumberID string `json:"phone_number_id" validate:"required"`
	BusinessID    string `json:"business_id,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	VerifyToken   string `json:"verify_token,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
	HTTPTimeout   int    `json:"http_timeout,omitempty"`
	MaxRetries    int    `json:"max_retries,omitempty"`
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
		config.APIVersion = whatsappAPIVersion
	}
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &WhatsAppProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		},
		baseURL: fmt.Sprintf("%s/%s/%s", whatsappAPIURL, config.APIVersion, config.PhoneNumberID),
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
		MessageID: response.Messages[0].ID,
		Provider:  whatsappProvider,
		To:        message.To,
		Status:    msgx.StatusPending,
		Timestamp: time.Now(),
		ProviderData: map[string]any{
			"whatsapp_id": response.Messages[0].ID,
			"wa_id":       response.Contacts[0].WaID,
		},
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

		// Add delay to respect rate limits (more conservative for v23.0)
		if i < len(messages)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return &msgx.BulkResponse{
		TotalSent:   totalSent,
		TotalFailed: len(failures),
		Responses:   responses,
		FailedItems: failures,
	}, nil
}

// GetStatus retrieves message status (WhatsApp doesn't have a direct status API)
func (w *WhatsAppProvider) GetStatus(ctx context.Context, messageID string) (*msgx.Status, error) {
	// WhatsApp relies on webhooks for status updates
	// This is a placeholder implementation
	return &msgx.Status{
		MessageID: messageID,
		Status:    msgx.StatusPending,
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateNumber validates a WhatsApp number
func (w *WhatsAppProvider) ValidateNumber(ctx context.Context, phoneNumber string) (*msgx.NumberValidation, error) {
	cleaned := w.cleanPhoneNumber(phoneNumber)

	// Basic validation first
	if !w.isValidPhoneFormat(cleaned) {
		return nil, msgx.Registry.New(msgx.ErrNumberValidationFailed).
			WithDetail("phone_number", phoneNumber).
			WithDetail("cleaned", cleaned).
			WithDetail("reason", "Invalid format").
			WithDetail("provider", whatsappProvider)
	}

	// WhatsApp doesn't have a direct validation API
	// We'll do basic format validation
	return &msgx.NumberValidation{
		PhoneNumber: cleaned,
		IsValid:     true,
		LineType:    "mobile",
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
	if config.VerifyToken != "" {
		w.config.VerifyToken = config.VerifyToken
	}

	// WhatsApp webhook setup is typically done through Meta Business Manager
	// This method validates the configuration
	return nil
}

// HandleWebhook processes incoming webhook requests
func (w *WhatsAppProvider) HandleWebhook(ctx context.Context, req *http.Request) (*msgx.IncomingMessage, error) {
	// Handle webhook verification challenge
	if req.Method == "GET" {
		return w.handleVerificationChallenge(req)
	}

	// Verify webhook signature
	if err := w.VerifyWebhook(req); err != nil {
		return nil, err
	}

	// Parse JSON body
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

	signature := req.Header.Get(whatsappSignatureHeader)
	if signature == "" {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Missing signature header")
	}

	// Read body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	// Restore body
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(w.config.WebhookSecret))
	mac.Write(body)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid signature")
	}

	return nil
}

// ParseIncomingMessage parses webhook data into structured message
func (w *WhatsAppProvider) ParseIncomingMessage(data []byte) (*msgx.IncomingMessage, error) {
	var webhook whatsappWebhookPayload
	if err := json.Unmarshal(data, &webhook); err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "unmarshal_json")
	}

	// Handle verification challenge
	if webhook.HubChallenge != "" {
		return nil, nil // This is handled separately
	}

	// Process webhook entries
	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			// Handle incoming messages
			for _, message := range change.Value.Messages {
				return w.convertWhatsAppMessage(message, change.Value.Metadata)
			}

			// Handle status updates
			for _, status := range change.Value.Statuses {
				// Status updates are handled separately
				_ = status
			}
		}
	}

	return nil, nil
}

// handleVerificationChallenge handles WhatsApp webhook verification
func (w *WhatsAppProvider) handleVerificationChallenge(req *http.Request) (*msgx.IncomingMessage, error) {
	verifyToken := req.URL.Query().Get("hub.verify_token")
	challenge := req.URL.Query().Get("hub.challenge")

	if w.config.VerifyToken != "" && verifyToken != w.config.VerifyToken {
		return nil, msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid verify token")
	}

	// Return challenge (this should be handled by the webhook server)
	_ = challenge

	return nil, nil
}

// ========== Helper Methods ==========

func (w *WhatsAppProvider) convertToWhatsAppMessage(msg msgx.Message) (*whatsappMessage, error) {
	whatsappMsg := &whatsappMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               w.cleanPhoneNumber(msg.To),
	}

	switch msg.Type {
	case msgx.MessageTypeText:
		if msg.Content.Text == nil {
			return nil, fmt.Errorf("text content is required for text messages")
		}
		whatsappMsg.Type = "text"
		whatsappMsg.Text = &whatsappTextMessage{
			Body:       msg.Content.Text.Body,
			PreviewURL: msg.Content.Text.PreviewURL,
		}

	case msgx.MessageTypeImage:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for image messages")
		}
		whatsappMsg.Type = "image"
		whatsappMsg.Image = &whatsappMediaMessage{
			Link:    msg.Content.Media.URL,
			Caption: msg.Content.Media.Caption,
		}

	case msgx.MessageTypeDocument:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for document messages")
		}
		whatsappMsg.Type = "document"
		whatsappMsg.Document = &whatsappDocumentMessage{
			Link:     msg.Content.Media.URL,
			Caption:  msg.Content.Media.Caption,
			Filename: msg.Content.Media.Filename,
		}

	case msgx.MessageTypeAudio:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for audio messages")
		}
		whatsappMsg.Type = "audio"
		whatsappMsg.Audio = &whatsappMediaMessage{
			Link: msg.Content.Media.URL,
		}

	case msgx.MessageTypeVideo:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for video messages")
		}
		whatsappMsg.Type = "video"
		whatsappMsg.Video = &whatsappMediaMessage{
			Link:    msg.Content.Media.URL,
			Caption: msg.Content.Media.Caption,
		}

	case msgx.MessageTypeTemplate:
		if msg.Content.Template == nil {
			return nil, fmt.Errorf("template content is required for template messages")
		}
		whatsappMsg.Type = "template"
		whatsappMsg.Template = &whatsappTemplateMessage{
			Name:     msg.Content.Template.Name,
			Language: whatsappLanguage{Code: msg.Content.Template.Language},
		}

		// Convert parameters (improved for v23.0)
		if len(msg.Content.Template.Parameters) > 0 {
			components := []whatsappTemplateComponent{
				{
					Type: "body",
				},
			}

			for key, value := range msg.Content.Template.Parameters {
				components[0].Parameters = append(components[0].Parameters, whatsappTemplateParameter{
					Type: "text",
					Text: fmt.Sprintf("%v", value),
				})
				_ = key // Avoid unused variable
			}

			whatsappMsg.Template.Components = components
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	return whatsappMsg, nil
}

func (w *WhatsAppProvider) sendMessage(ctx context.Context, message *whatsappMessage) (*whatsappSendResponse, error) {
	url := fmt.Sprintf("%s/messages", w.baseURL)

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

	// WhatsApp API returns 200 for successful sends in v23.0
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
				WithDetail("whatsapp_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusServiceUnavailable:
			return msgx.Registry.New(msgx.ErrProviderUnavailable).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusUnauthorized:
			return msgx.Registry.New(msgx.ErrProviderConfigInvalid).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp).
				WithDetail("reason", "Invalid access token")
		case http.StatusBadRequest:
			return msgx.Registry.New(msgx.ErrInvalidMessage).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		default:
			return msgx.Registry.New(msgx.ErrSendFailed).
				WithDetail("provider", whatsappProvider).
				WithDetail("whatsapp_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		}
	}

	return msgx.Registry.New(msgx.ErrSendFailed).
		WithDetail("provider", whatsappProvider).
		WithDetail("http_status", resp.StatusCode).
		WithDetail("response_body", string(body))
}

func (w *WhatsAppProvider) convertWhatsAppMessage(message whatsappIncomingMessage, metadata whatsappMetadata) (*msgx.IncomingMessage, error) {
	incomingMsg := &msgx.IncomingMessage{
		ID:        message.ID,
		Provider:  whatsappProvider,
		From:      message.From,
		To:        metadata.PhoneNumberID,
		Timestamp: time.Unix(message.Timestamp, 0),
		Type:      msgx.MessageTypeText, // Default
		RawData:   map[string]any{"whatsapp_message": message},
	}

	// Parse message content based on type
	switch message.Type {
	case "text":
		incomingMsg.Type = msgx.MessageTypeText
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: message.Text.Body,
		}

	case "image":
		incomingMsg.Type = msgx.MessageTypeImage
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			Caption:  message.Image.Caption,
			MimeType: message.Image.MimeType,
		}
		// Note: WhatsApp media URLs need to be downloaded separately

	case "document":
		incomingMsg.Type = msgx.MessageTypeDocument
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			Caption:  message.Document.Caption,
			Filename: message.Document.Filename,
			MimeType: message.Document.MimeType,
		}

	case "audio":
		incomingMsg.Type = msgx.MessageTypeAudio
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			MimeType: message.Audio.MimeType,
		}

	case "video":
		incomingMsg.Type = msgx.MessageTypeVideo
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			Caption:  message.Video.Caption,
			MimeType: message.Video.MimeType,
		}

	case "location":
		incomingMsg.Content.Location = &msgx.LocationContent{
			Latitude:  message.Location.Latitude,
			Longitude: message.Location.Longitude,
			Name:      message.Location.Name,
			Address:   message.Location.Address,
		}

	case "contacts":
		if len(message.Contacts) > 0 {
			contact := message.Contacts[0]
			incomingMsg.Content.Contact = &msgx.ContactContent{
				Name: contact.Name.FormattedName,
			}
			if len(contact.Phones) > 0 {
				incomingMsg.Content.Contact.PhoneNumber = contact.Phones[0].Phone
			}
		}

	// New message types in v23.0
	case "reaction":
		// Handle message reactions (new in recent versions)
		incomingMsg.Type = msgx.MessageTypeText
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: "User reacted to a message", // Simplified handling
		}

	case "button":
		// Handle button responses
		incomingMsg.Type = msgx.MessageTypeText
		// Button handling would need additional parsing

	case "interactive":
		// Handle interactive message responses
		incomingMsg.Type = msgx.MessageTypeText
		// Interactive handling would need additional parsing
	}

	return incomingMsg, nil
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

	// Add + if not present and looks like international number
	if !strings.HasPrefix(cleaned, "+") && len(cleaned) > 10 {
		cleaned = "+" + cleaned
	}

	return cleaned
}

func (w *WhatsAppProvider) isValidPhoneFormat(phoneNumber string) bool {
	// Basic E.164 format validation
	e164Regex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return e164Regex.MatchString(phoneNumber)
}

// ========== WhatsApp API Structures ==========

// Send message structures
type whatsappMessage struct {
	MessagingProduct string                   `json:"messaging_product"`
	RecipientType    string                   `json:"recipient_type"`
	To               string                   `json:"to"`
	Type             string                   `json:"type"`
	Text             *whatsappTextMessage     `json:"text,omitempty"`
	Image            *whatsappMediaMessage    `json:"image,omitempty"`
	Document         *whatsappDocumentMessage `json:"document,omitempty"`
	Audio            *whatsappMediaMessage    `json:"audio,omitempty"`
	Video            *whatsappMediaMessage    `json:"video,omitempty"`
	Template         *whatsappTemplateMessage `json:"template,omitempty"`
}

type whatsappTextMessage struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type whatsappMediaMessage struct {
	Link    string `json:"link,omitempty"`
	Caption string `json:"caption,omitempty"`
}

type whatsappDocumentMessage struct {
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type whatsappTemplateMessage struct {
	Name       string                      `json:"name"`
	Language   whatsappLanguage            `json:"language"`
	Components []whatsappTemplateComponent `json:"components,omitempty"`
}

type whatsappLanguage struct {
	Code string `json:"code"`
}

type whatsappTemplateComponent struct {
	Type       string                      `json:"type"`
	Parameters []whatsappTemplateParameter `json:"parameters,omitempty"`
}

type whatsappTemplateParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Response structures
type whatsappSendResponse struct {
	MessagingProduct string                    `json:"messaging_product"`
	Contacts         []whatsappContact         `json:"contacts"`
	Messages         []whatsappMessageResponse `json:"messages"`
}

type whatsappContact struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type whatsappMessageResponse struct {
	ID string `json:"id"`
}

type whatsappErrorResponse struct {
	Error whatsappError `json:"error"`
}

type whatsappError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode"`
	FbtraceID    string `json:"fbtrace_id"`
}

// Webhook structures
type whatsappWebhookPayload struct {
	Object       string                 `json:"object"`
	Entry        []whatsappWebhookEntry `json:"entry"`
	HubMode      string                 `json:"hub.mode,omitempty"`
	HubChallenge string                 `json:"hub.challenge,omitempty"`
	HubVerify    string                 `json:"hub.verify_token,omitempty"`
}

type whatsappWebhookEntry struct {
	ID      string                  `json:"id"`
	Changes []whatsappWebhookChange `json:"changes"`
}

type whatsappWebhookChange struct {
	Value whatsappWebhookValue `json:"value"`
	Field string               `json:"field"`
}

type whatsappWebhookValue struct {
	MessagingProduct string                    `json:"messaging_product"`
	Metadata         whatsappMetadata          `json:"metadata"`
	Contacts         []whatsappWebhookContact  `json:"contacts,omitempty"`
	Messages         []whatsappIncomingMessage `json:"messages,omitempty"`
	Statuses         []whatsappStatusUpdate    `json:"statuses,omitempty"`
}

type whatsappMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type whatsappWebhookContact struct {
	Profile whatsappProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type whatsappProfile struct {
	Name string `json:"name"`
}

// Incoming message structures
type whatsappIncomingMessage struct {
	From      string                    `json:"from"`
	ID        string                    `json:"id"`
	Timestamp int64                     `json:"timestamp"`
	Type      string                    `json:"type"`
	Context   *whatsappMessageContext   `json:"context,omitempty"`
	Text      *whatsappIncomingText     `json:"text,omitempty"`
	Image     *whatsappIncomingMedia    `json:"image,omitempty"`
	Document  *whatsappIncomingDocument `json:"document,omitempty"`
	Audio     *whatsappIncomingMedia    `json:"audio,omitempty"`
	Video     *whatsappIncomingMedia    `json:"video,omitempty"`
	Location  *whatsappIncomingLocation `json:"location,omitempty"`
	Contacts  []whatsappIncomingContact `json:"contacts,omitempty"`
}

type whatsappMessageContext struct {
	From     string `json:"from"`
	ID       string `json:"id"`
	Referred struct {
		Product struct {
			CatalogID         string `json:"catalog_id"`
			ProductRetailerID string `json:"product_retailer_id"`
		} `json:"product"`
	} `json:"referred,omitempty"`
}

type whatsappIncomingText struct {
	Body string `json:"body"`
}

type whatsappIncomingMedia struct {
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
	ID       string `json:"id"`
}

type whatsappIncomingDocument struct {
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
	ID       string `json:"id"`
}

type whatsappIncomingLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type whatsappIncomingContact struct {
	Addresses []whatsappContactAddress `json:"addresses,omitempty"`
	Birthday  string                   `json:"birthday,omitempty"`
	Emails    []whatsappContactEmail   `json:"emails,omitempty"`
	Name      whatsappContactName      `json:"name"`
	Org       whatsappContactOrg       `json:"org,omitempty"`
	Phones    []whatsappContactPhone   `json:"phones,omitempty"`
	URLs      []whatsappContactURL     `json:"urls,omitempty"`
}

type whatsappContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

type whatsappContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type whatsappContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

type whatsappContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type whatsappContactPhone struct {
	Phone string `json:"phone,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
	Type  string `json:"type,omitempty"`
}

type whatsappContactURL struct {
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

// Status update structures
type whatsappStatusUpdate struct {
	ID           string                `json:"id"`
	Status       string                `json:"status"`
	Timestamp    string                `json:"timestamp"`
	RecipientID  string                `json:"recipient_id"`
	Conversation *whatsappConversation `json:"conversation,omitempty"`
	Pricing      *whatsappPricing      `json:"pricing,omitempty"`
}

type whatsappConversation struct {
	ID                  string                     `json:"id"`
	ExpirationTimestamp string                     `json:"expiration_timestamp,omitempty"`
	Origin              whatsappConversationOrigin `json:"origin"`
}

type whatsappConversationOrigin struct {
	Type string `json:"type"`
}

type whatsappPricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

