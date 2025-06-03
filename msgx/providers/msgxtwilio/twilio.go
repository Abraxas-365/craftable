package msgxtwilio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/msgx"
)

const (
	twilioAPIURL          = "https://api.twilio.com"
	twilioProvider        = "twilio"
	twilioSignatureHeader = "X-Twilio-Signature"
)

// TwilioConfig holds Twilio API configuration
type TwilioConfig struct {
	AccountSID    string `json:"account_sid" validate:"required"`
	AuthToken     string `json:"auth_token" validate:"required"`
	FromNumber    string `json:"from_number" validate:"required"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
	HTTPTimeout   int    `json:"http_timeout,omitempty"`
}

// TwilioProvider implements the msgx.Provider interface
type TwilioProvider struct {
	config     TwilioConfig
	httpClient *http.Client
	baseURL    string
}

// NewTwilioProvider creates a new Twilio provider
func NewTwilioProvider(config TwilioConfig) *TwilioProvider {
	if config.APIVersion == "" {
		config.APIVersion = "2010-04-01"
	}
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30
	}

	return &TwilioProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		},
		baseURL: fmt.Sprintf("%s/%s/Accounts/%s", twilioAPIURL, config.APIVersion, config.AccountSID),
	}
}

// ========== Sender Interface Implementation ==========

// Send sends a message via Twilio API
func (t *TwilioProvider) Send(ctx context.Context, message msgx.Message) (*msgx.Response, error) {
	// Convert to Twilio API format
	twilioMsg, err := t.convertToTwilioMessage(message)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrInvalidMessage).
			WithCause(err).
			WithDetail("message", message).
			WithDetail("provider", twilioProvider)
	}

	// Send via API
	response, err := t.sendMessage(ctx, twilioMsg)
	if err != nil {
		return nil, err
	}

	return &msgx.Response{
		MessageID: response.SID,
		Provider:  twilioProvider,
		To:        message.To,
		Status:    t.convertTwilioStatus(response.Status),
		Timestamp: time.Now(),
		ProviderData: map[string]any{
			"twilio_sid":    response.SID,
			"twilio_status": response.Status,
			"price":         response.Price,
			"price_unit":    response.PriceUnit,
		},
	}, nil
}

// SendBulk sends multiple messages
func (t *TwilioProvider) SendBulk(ctx context.Context, messages []msgx.Message) (*msgx.BulkResponse, error) {
	responses := make([]msgx.Response, 0, len(messages))
	failures := make([]msgx.BulkFailure, 0)
	totalSent := 0

	for i, message := range messages {
		response, err := t.Send(ctx, message)
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
func (t *TwilioProvider) GetStatus(ctx context.Context, messageID string) (*msgx.Status, error) {
	url := fmt.Sprintf("%s/Messages/%s.json", t.baseURL, messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("operation", "get_status").
			WithDetail("message_id", messageID).
			WithDetail("provider", twilioProvider)
	}

	// Add Basic Auth
	req.SetBasicAuth(t.config.AccountSID, t.config.AuthToken)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("message_id", messageID).
			WithDetail("operation", "http_request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, t.handleAPIError(resp)
	}

	var statusResp twilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("operation", "decode_status_response").
			WithDetail("provider", twilioProvider)
	}

	return &msgx.Status{
		MessageID: messageID,
		Status:    t.convertTwilioStatus(statusResp.Status),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateNumber validates a phone number using Twilio Lookup API
func (t *TwilioProvider) ValidateNumber(ctx context.Context, phoneNumber string) (*msgx.NumberValidation, error) {
	cleaned := t.cleanPhoneNumber(phoneNumber)

	// Basic validation first
	if !t.isValidPhoneFormat(cleaned) {
		return nil, msgx.Registry.New(msgx.ErrNumberValidationFailed).
			WithDetail("phone_number", phoneNumber).
			WithDetail("cleaned", cleaned).
			WithDetail("reason", "Invalid format").
			WithDetail("provider", twilioProvider)
	}

	// Use Twilio Lookup API for advanced validation
	lookupURL := fmt.Sprintf("https://lookups.twilio.com/v1/PhoneNumbers/%s", url.QueryEscape(cleaned))

	req, err := http.NewRequestWithContext(ctx, "GET", lookupURL, nil)
	if err != nil {
		return &msgx.NumberValidation{
			PhoneNumber: cleaned,
			IsValid:     true, // Fallback to basic validation
			LineType:    "unknown",
		}, nil
	}

	req.SetBasicAuth(t.config.AccountSID, t.config.AuthToken)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &msgx.NumberValidation{
			PhoneNumber: cleaned,
			IsValid:     true, // Fallback to basic validation
			LineType:    "unknown",
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var lookupResp twilioLookupResponse
		if err := json.NewDecoder(resp.Body).Decode(&lookupResp); err == nil {
			return &msgx.NumberValidation{
				PhoneNumber: lookupResp.PhoneNumber,
				IsValid:     true,
				LineType:    "mobile", // SMS assumes mobile
				Country:     lookupResp.CountryCode,
			}, nil
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, msgx.Registry.New(msgx.ErrNumberValidationFailed).
			WithDetail("phone_number", phoneNumber).
			WithDetail("reason", "Number not found").
			WithDetail("provider", twilioProvider)
	}

	// Fallback to basic validation
	return &msgx.NumberValidation{
		PhoneNumber: cleaned,
		IsValid:     true,
		LineType:    "mobile",
	}, nil
}

// GetProviderName returns the provider name
func (t *TwilioProvider) GetProviderName() string {
	return twilioProvider
}

// ========== Receiver Interface Implementation ==========

// SetupWebhook configures the webhook endpoint
func (t *TwilioProvider) SetupWebhook(config msgx.WebhookConfig) error {
	// Store webhook config for verification
	if config.Secret != "" {
		t.config.WebhookSecret = config.Secret
	}

	// Twilio webhook setup is typically done through the console or API
	// This method validates the configuration
	return nil
}

// HandleWebhook processes incoming webhook requests
func (t *TwilioProvider) HandleWebhook(ctx context.Context, req *http.Request) (*msgx.IncomingMessage, error) {
	// Verify webhook signature
	if err := t.VerifyWebhook(req); err != nil {
		return nil, err
	}

	// Parse form data
	if err := req.ParseForm(); err != nil {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "parse_form")
	}

	return t.ParseIncomingMessage(req.Form)
}

// VerifyWebhook verifies the webhook signature
func (t *TwilioProvider) VerifyWebhook(req *http.Request) error {
	if t.config.WebhookSecret == "" {
		return nil // Skip verification if no secret configured
	}

	signature := req.Header.Get(twilioSignatureHeader)
	if signature == "" {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", twilioProvider).
			WithDetail("reason", "Missing signature header")
	}

	// Read and restore body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "read_body")
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Parse form to get the URL for signature validation
	if err := req.ParseForm(); err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "parse_form")
	}

	// Restore body again after parsing
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Build the URL string for signature verification
	urlStr := t.buildURLForSignature(req)

	// Calculate expected signature
	mac := hmac.New(sha1.New, []byte(t.config.WebhookSecret))
	mac.Write([]byte(urlStr))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", twilioProvider).
			WithDetail("reason", "Invalid signature")
	}

	return nil
}

// ParseIncomingMessage parses webhook data into structured message
func (t *TwilioProvider) ParseIncomingMessage(form url.Values) (*msgx.IncomingMessage, error) {
	messageSID := form.Get("MessageSid")
	if messageSID == "" {
		return nil, msgx.Registry.New(msgx.ErrWebhookParseFailed).
			WithDetail("provider", twilioProvider).
			WithDetail("reason", "Missing MessageSid")
	}

	// Check if this is a status callback
	messageStatus := form.Get("MessageStatus")
	if messageStatus != "" && form.Get("Body") == "" {
		// This is a status update, not an incoming message
		return nil, nil
	}

	incomingMsg := &msgx.IncomingMessage{
		ID:        messageSID,
		Provider:  twilioProvider,
		From:      form.Get("From"),
		To:        form.Get("To"),
		Timestamp: time.Now(),
		Type:      msgx.MessageTypeText,
		RawData:   map[string]any{"twilio_form": form},
	}

	// Parse message content
	body := form.Get("Body")
	if body != "" {
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: body,
		}
	}

	// Check for media
	numMedia := form.Get("NumMedia")
	if numMedia != "" && numMedia != "0" {
		// Handle media messages
		mediaURL := form.Get("MediaUrl0")
		mediaType := form.Get("MediaContentType0")

		if mediaURL != "" {
			// Determine message type from content type
			switch {
			case strings.HasPrefix(mediaType, "image/"):
				incomingMsg.Type = msgx.MessageTypeImage
			case strings.HasPrefix(mediaType, "video/"):
				incomingMsg.Type = msgx.MessageTypeVideo
			case strings.HasPrefix(mediaType, "audio/"):
				incomingMsg.Type = msgx.MessageTypeAudio
			default:
				incomingMsg.Type = msgx.MessageTypeDocument
			}

			incomingMsg.Content.Media = &msgx.IncomingMediaContent{
				URL:      mediaURL,
				MimeType: mediaType,
				Caption:  body, // Twilio includes text as caption for media
			}
		}
	}

	return incomingMsg, nil
}

// ========== Helper Methods ==========

func (t *TwilioProvider) convertToTwilioMessage(msg msgx.Message) (*twilioMessage, error) {
	twilioMsg := &twilioMessage{
		To:   t.cleanPhoneNumber(msg.To),
		From: t.config.FromNumber,
	}

	switch msg.Type {
	case msgx.MessageTypeText:
		if msg.Content.Text == nil {
			return nil, fmt.Errorf("text content is required for text messages")
		}
		twilioMsg.Body = msg.Content.Text.Body

	case msgx.MessageTypeImage, msgx.MessageTypeDocument, msgx.MessageTypeAudio, msgx.MessageTypeVideo:
		if msg.Content.Media == nil {
			return nil, fmt.Errorf("media content is required for media messages")
		}

		twilioMsg.MediaURL = []string{msg.Content.Media.URL}
		if msg.Content.Media.Caption != "" {
			twilioMsg.Body = msg.Content.Media.Caption
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	return twilioMsg, nil
}

func (t *TwilioProvider) sendMessage(ctx context.Context, message *twilioMessage) (*twilioMessageResponse, error) {
	ur := fmt.Sprintf("%s/Messages.json", t.baseURL)

	// Prepare form data
	data := url.Values{}
	data.Set("To", message.To)
	data.Set("From", message.From)
	if message.Body != "" {
		data.Set("Body", message.Body)
	}
	for i, mediaURL := range message.MediaURL {
		data.Set(fmt.Sprintf("MediaUrl.%d", i), mediaURL)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ur, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "create_request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.config.AccountSID, t.config.AuthToken)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "http_request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, t.handleAPIError(resp)
	}

	var sendResp twilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", twilioProvider).
			WithDetail("operation", "decode_response")
	}

	return &sendResp, nil
}

func (t *TwilioProvider) handleAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errorResp twilioErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Code != 0 {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return msgx.Registry.New(msgx.ErrRateLimitExceeded).
				WithDetail("provider", twilioProvider).
				WithDetail("twilio_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusServiceUnavailable:
			return msgx.Registry.New(msgx.ErrProviderUnavailable).
				WithDetail("provider", twilioProvider).
				WithDetail("twilio_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		case http.StatusUnauthorized:
			return msgx.Registry.New(msgx.ErrProviderConfigInvalid).
				WithDetail("provider", twilioProvider).
				WithDetail("twilio_error", errorResp).
				WithDetail("reason", "Invalid credentials")
		case http.StatusBadRequest:
			return msgx.Registry.New(msgx.ErrInvalidMessage).
				WithDetail("provider", twilioProvider).
				WithDetail("twilio_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		default:
			return msgx.Registry.New(msgx.ErrSendFailed).
				WithDetail("provider", twilioProvider).
				WithDetail("twilio_error", errorResp).
				WithDetail("http_status", resp.StatusCode)
		}
	}

	return msgx.Registry.New(msgx.ErrSendFailed).
		WithDetail("provider", twilioProvider).
		WithDetail("http_status", resp.StatusCode).
		WithDetail("response_body", string(body))
}

func (t *TwilioProvider) cleanPhoneNumber(phoneNumber string) string {
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

func (t *TwilioProvider) isValidPhoneFormat(phoneNumber string) bool {
	// Basic E.164 format validation
	e164Regex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return e164Regex.MatchString(phoneNumber)
}

func (t *TwilioProvider) convertTwilioStatus(status string) msgx.MessageStatus {
	switch strings.ToLower(status) {
	case "queued", "accepted":
		return msgx.StatusPending
	case "sending", "sent":
		return msgx.StatusSent
	case "delivered":
		return msgx.StatusDelivered
	case "read":
		return msgx.StatusRead
	case "failed", "undelivered":
		return msgx.StatusFailed
	default:
		return msgx.StatusPending
	}
}

func (t *TwilioProvider) buildURLForSignature(req *http.Request) string {
	// Build URL string for Twilio signature verification
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}

	baseURL := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.Path)

	// Sort form parameters and append them
	var params []string
	for key, values := range req.Form {
		for _, value := range values {
			params = append(params, key+value)
		}
	}

	return baseURL + strings.Join(params, "")
}

// ========== Twilio API Structures ==========

type twilioMessage struct {
	To       string   `json:"to"`
	From     string   `json:"from"`
	Body     string   `json:"body,omitempty"`
	MediaURL []string `json:"media_url,omitempty"`
}

type twilioMessageResponse struct {
	SID         string      `json:"sid"`
	AccountSID  string      `json:"account_sid"`
	From        string      `json:"from"`
	To          string      `json:"to"`
	Body        string      `json:"body"`
	Status      string      `json:"status"`
	Direction   string      `json:"direction"`
	Price       string      `json:"price"`
	PriceUnit   string      `json:"price_unit"`
	DateCreated *TwilioTime `json:"date_created"`
	DateUpdated *TwilioTime `json:"date_updated"`
	DateSent    *TwilioTime `json:"date_sent"`
	URI         string      `json:"uri"`
}

// TwilioTime handles Twilio's RFC1123 date format
type TwilioTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for TwilioTime
func (tt *TwilioTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := string(data[1 : len(data)-1])

	// Handle null/empty values
	if str == "" || str == "null" {
		return nil
	}

	// Parse RFC1123 format (what Twilio uses)
	t, err := time.Parse(time.RFC1123, str)
	if err != nil {
		// Try RFC1123Z format as fallback
		t, err = time.Parse(time.RFC1123Z, str)
		if err != nil {
			return err
		}
	}

	tt.Time = t
	return nil
}

// MarshalJSON implements json.Marshaler for TwilioTime
func (tt TwilioTime) MarshalJSON() ([]byte, error) {
	if tt.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + tt.Time.Format(time.RFC1123) + `"`), nil
}

type twilioErrorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

type twilioLookupResponse struct {
	CallerName     any    `json:"caller_name"`
	CarrierName    string `json:"carrier_name"`
	CountryCode    string `json:"country_code"`
	PhoneNumber    string `json:"phone_number"`
	AddOns         any    `json:"add_ons"`
	URL            string `json:"url"`
	NationalFormat string `json:"national_format"`
}
