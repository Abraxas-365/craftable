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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/logx"
	"github.com/Abraxas-365/craftable/msgx"
)

const (
	whatsappAPIURL          = "https://graph.facebook.com"
	whatsappProvider        = "whatsapp"
	whatsappSignatureHeader = "X-Hub-Signature-256"
	whatsappAPIVersion      = "v23.0"
)

// ========== Template API Structures ==========

// TemplateFromAPI represents template structure from WhatsApp API
type TemplateFromAPI struct {
	Name            string                     `json:"name"`
	Language        string                     `json:"language"`
	Status          string                     `json:"status"`
	Category        string                     `json:"category"`
	ID              string                     `json:"id"`
	ParameterFormat string                     `json:"parameter_format,omitempty"`
	Components      []TemplateComponentFromAPI `json:"components"`
}

type TemplateComponentFromAPI struct {
	Type    string           `json:"type"`             // HEADER, BODY, FOOTER, BUTTONS
	Format  string           `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT
	Text    string           `json:"text,omitempty"`
	Example *TemplateExample `json:"example,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

type TemplateExample struct {
	HeaderText          []string             `json:"header_text,omitempty"`
	BodyText            [][]string           `json:"body_text,omitempty"`
	BodyTextNamedParams []BodyTextNamedParam `json:"body_text_named_params,omitempty"`
}

type BodyTextNamedParam struct {
	ParamName string `json:"param_name"`
	Example   string `json:"example"`
}

type TemplateButton struct {
	Type string `json:"type"`
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// TemplateCache holds cached template data
type TemplateCache struct {
	Template  TemplateFromAPI `json:"template"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// ========== Configuration ==========

// WhatsAppConfig holds WhatsApp Business API configuration
type WhatsAppConfig struct {
	AccessToken       string `json:"access_token" validate:"required"`
	PhoneNumberID     string `json:"phone_number_id" validate:"required"`
	BusinessAccountID string `json:"business_account_id" validate:"required"` // Required for template API
	WebhookSecret     string `json:"webhook_secret,omitempty"`
	VerifyToken       string `json:"verify_token,omitempty"`
	APIVersion        string `json:"api_version,omitempty"`
	HTTPTimeout       int    `json:"http_timeout,omitempty"`
	MaxRetries        int    `json:"max_retries,omitempty"`
	CacheTemplates    bool   `json:"cache_templates,omitempty"`    // Cache templates to avoid repeated API calls
	TemplateCacheTTL  int    `json:"template_cache_ttl,omitempty"` // Cache TTL in minutes
}

// WhatsAppProvider implements the msgx.Provider interface
type WhatsAppProvider struct {
	config         WhatsAppConfig
	httpClient     *http.Client
	baseURL        string
	businessAPIURL string
	templateCache  map[string]TemplateCache
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
	if config.TemplateCacheTTL == 0 {
		config.TemplateCacheTTL = 60 // 1 hour default
	}

	return &WhatsAppProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		},
		baseURL:        fmt.Sprintf("%s/%s/%s", whatsappAPIURL, config.APIVersion, config.PhoneNumberID),
		businessAPIURL: fmt.Sprintf("%s/%s/%s", whatsappAPIURL, config.APIVersion, config.BusinessAccountID),
		templateCache:  make(map[string]TemplateCache),
	}
}

// ========== Template API Methods ==========

// GetTemplate fetches template from WhatsApp API
func (w *WhatsAppProvider) GetTemplate(ctx context.Context, templateName, language string) (*TemplateFromAPI, error) {
	// Check cache first
	if w.config.CacheTemplates {
		cacheKey := fmt.Sprintf("%s_%s", templateName, language)
		if cached, exists := w.templateCache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
			return &cached.Template, nil
		}
	}

	// Fetch from API
	url := fmt.Sprintf("%s/message_templates?name=%s&language=%s", w.businessAPIURL, templateName, language)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var templateResponse struct {
		Data []TemplateFromAPI `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&templateResponse); err != nil {
		return nil, fmt.Errorf("failed to decode template response: %w", err)
	}

	if len(templateResponse.Data) == 0 {
		return nil, fmt.Errorf("template not found: %s_%s", templateName, language)
	}

	template := templateResponse.Data[0]

	// Cache the template
	if w.config.CacheTemplates {
		cacheKey := fmt.Sprintf("%s_%s", templateName, language)
		w.templateCache[cacheKey] = TemplateCache{
			Template:  template,
			ExpiresAt: time.Now().Add(time.Duration(w.config.TemplateCacheTTL) * time.Minute),
		}
	}

	return &template, nil
}

// extractParametersFromText extracts parameters from template text in order of appearance
func (w *WhatsAppProvider) extractParametersFromText(templateText string, parameterValues map[string]any) []whatsappTemplateParameter {
	if templateText == "" || len(parameterValues) == 0 {
		return nil
	}

	// Extract variables in order of appearance
	re := regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	matches := re.FindAllStringSubmatch(templateText, -1)

	var parameters []whatsappTemplateParameter
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			variable := strings.TrimSpace(match[1])
			if !seen[variable] {
				if value, exists := parameterValues[variable]; exists {
					parameters = append(parameters, whatsappTemplateParameter{
						Type: "text",
						Text: fmt.Sprintf("%v", value),
					})
					seen[variable] = true
				}
			}
		}
	}

	return parameters
}

// buildTemplateComponentsForNamed handles NAMED parameter templates
func (w *WhatsAppProvider) buildTemplateComponentsForNamed(ctx context.Context, templateContent *msgx.TemplateContent) ([]whatsappTemplateComponent, error) {
	// For NAMED parameter templates, we need to fetch the template structure
	template, err := w.GetTemplate(ctx, templateContent.Name, templateContent.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template structure: %w", err)
	}

	var components []whatsappTemplateComponent

	// Process each component from the template
	for _, apiComponent := range template.Components {
		switch apiComponent.Type {
		case "HEADER":
			if apiComponent.Format == "TEXT" && apiComponent.Text != "" {
				headerParams := w.extractParametersFromText(apiComponent.Text, templateContent.Parameters)
				if len(headerParams) > 0 {
					component := whatsappTemplateComponent{
						Type:       "header",
						Parameters: headerParams,
					}
					components = append(components, component)
				}
			}

		case "BODY":
			if apiComponent.Text != "" {
				bodyParams := w.extractParametersFromText(apiComponent.Text, templateContent.Parameters)
				if len(bodyParams) > 0 {
					component := whatsappTemplateComponent{
						Type:       "body",
						Parameters: bodyParams,
					}
					components = append(components, component)
				}
			}
		}
	}

	return components, nil
}

// buildTemplateComponentsForPositional handles numbered/positional parameter templates
func (w *WhatsAppProvider) buildTemplateComponentsForPositional(ctx context.Context, templateContent *msgx.TemplateContent) ([]whatsappTemplateComponent, error) {
	// Get template structure to count parameters per component
	template, err := w.GetTemplate(ctx, templateContent.Name, templateContent.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template structure: %w", err)
	}

	var components []whatsappTemplateComponent
	parameterIndex := 0

	// Convert parameters map to ordered slice for consistent parameter assignment
	parameterValues := make([]string, 0, len(templateContent.Parameters))

	// Sort parameter keys to ensure consistent ordering
	var sortedKeys []string
	for key := range templateContent.Parameters {
		sortedKeys = append(sortedKeys, key)
	}

	// Try to sort numerically if keys are numbers, otherwise sort alphabetically
	sort.Slice(sortedKeys, func(i, j int) bool {
		numI, errI := strconv.Atoi(sortedKeys[i])
		numJ, errJ := strconv.Atoi(sortedKeys[j])
		if errI == nil && errJ == nil {
			return numI < numJ
		}
		return sortedKeys[i] < sortedKeys[j]
	})

	for _, key := range sortedKeys {
		parameterValues = append(parameterValues, fmt.Sprintf("%v", templateContent.Parameters[key]))
	}

	// Process each component from the template
	for _, apiComponent := range template.Components {
		switch apiComponent.Type {
		case "HEADER":
			if apiComponent.Format == "TEXT" {
				// Count parameters in header text
				headerParamCount := w.countTemplateParameters(apiComponent.Text)
				if headerParamCount > 0 {
					component := whatsappTemplateComponent{
						Type: "header",
					}

					// Add parameters for header
					for i := 0; i < headerParamCount && parameterIndex < len(parameterValues); i++ {
						component.Parameters = append(component.Parameters, whatsappTemplateParameter{
							Type: "text",
							Text: parameterValues[parameterIndex],
						})
						parameterIndex++
					}
					components = append(components, component)
				}
			}

		case "BODY":
			// Count parameters in body text
			bodyParamCount := w.countTemplateParameters(apiComponent.Text)
			if bodyParamCount > 0 {
				component := whatsappTemplateComponent{
					Type: "body",
				}

				// Add parameters for body
				for i := 0; i < bodyParamCount && parameterIndex < len(parameterValues); i++ {
					component.Parameters = append(component.Parameters, whatsappTemplateParameter{
						Type: "text",
						Text: parameterValues[parameterIndex],
					})
					parameterIndex++
				}
				components = append(components, component)
			}
		}
	}

	return components, nil
}

// countTemplateParameters counts template parameters in text
func (w *WhatsAppProvider) countTemplateParameters(text string) int {
	// Count {{1}}, {{2}}, etc. patterns
	re := regexp.MustCompile(`\{\{(\d+)\}\}`)
	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return 0
	}

	// Find the highest parameter number
	maxParam := 0
	for _, match := range matches {
		if len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil && num > maxParam {
				maxParam = num
			}
		}
	}

	return maxParam
}

// ResolveTemplateFromAPI resolves template content using API-fetched template
func (w *WhatsAppProvider) ResolveTemplateFromAPI(ctx context.Context, templateContent *msgx.TemplateContent) (*msgx.ResolvedContent, error) {
	// Fetch template from API
	template, err := w.GetTemplate(ctx, templateContent.Name, templateContent.Language)
	if err != nil {
		// Return basic resolved content if API fetch fails
		return &msgx.ResolvedContent{
			TemplateName:    templateContent.Name,
			Language:        templateContent.Language,
			Parameters:      templateContent.Parameters,
			ParameterCount:  len(templateContent.Parameters),
			ResolvedMessage: fmt.Sprintf("Template: %s (Language: %s) [API fetch failed: %v]", templateContent.Name, templateContent.Language, err),
		}, nil
	}

	resolved := &msgx.ResolvedContent{
		TemplateName:   templateContent.Name,
		Language:       templateContent.Language,
		Parameters:     templateContent.Parameters,
		ParameterCount: len(templateContent.Parameters),
	}

	// Process template components
	var header, body, footer string
	var resolvedHeader, resolvedBody, resolvedFooter string

	for _, component := range template.Components {
		switch component.Type {
		case "HEADER":
			header = component.Text
			resolvedHeader = w.resolveTemplateText(component.Text, templateContent.Parameters)
		case "BODY":
			body = component.Text
			resolvedBody = w.resolveTemplateText(component.Text, templateContent.Parameters)
		case "FOOTER":
			footer = component.Text
			resolvedFooter = component.Text // Footer usually doesn't have parameters
		}
	}

	// Set resolved content
	resolved.Header = header
	resolved.OriginalBody = body
	resolved.Footer = footer
	resolved.ResolvedBody = resolvedBody

	// Create complete resolved message
	var fullMessage strings.Builder
	if resolvedHeader != "" {
		fullMessage.WriteString(resolvedHeader + "\n\n")
	}
	if resolvedBody != "" {
		fullMessage.WriteString(resolvedBody)
	}
	if resolvedFooter != "" {
		fullMessage.WriteString("\n\n" + resolvedFooter)
	}

	resolved.ResolvedMessage = fullMessage.String()

	return resolved, nil
}

// resolveTemplateText replaces parameters in template text
func (w *WhatsAppProvider) resolveTemplateText(text string, parameters map[string]any) string {
	if len(parameters) == 0 {
		return text
	}

	resolved := text
	for key, value := range parameters {
		// Replace both numbered parameters {{1}}, {{2}} and named parameters {{name}}, {{career}}
		placeholder := fmt.Sprintf("{{%s}}", key)
		resolved = strings.ReplaceAll(resolved, placeholder, fmt.Sprintf("%v", value))
	}

	return resolved
}

// ========== Sender Interface Implementation ==========

// Send sends a message via WhatsApp Business API
func (w *WhatsAppProvider) Send(ctx context.Context, message msgx.Message) (*msgx.Response, error) {
	// Convert to WhatsApp API format
	whatsappMsg, err := w.convertToWhatsAppMessage(ctx, message)
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

	// Build the response
	msgxResponse := &msgx.Response{
		MessageID: response.Messages[0].ID,
		Provider:  whatsappProvider,
		To:        message.To,
		Status:    msgx.StatusPending,
		Timestamp: time.Now(),
		ProviderData: map[string]any{
			"whatsapp_id": response.Messages[0].ID,
			"wa_id":       response.Contacts[0].WaID,
		},
	}

	// For template messages, fetch template from API and resolve content
	if message.Type == msgx.MessageTypeTemplate && message.Content.Template != nil {
		resolvedContent, err := w.ResolveTemplateFromAPI(ctx, message.Content.Template)
		if err == nil {
			msgxResponse.ResolvedContent = resolvedContent
		} else {
			// Add error info to provider data
			msgxResponse.ProviderData["template_resolution_error"] = err.Error()
		}
	}

	return msgxResponse, nil
}

// convertToWhatsAppMessage converts msgx.Message to WhatsApp API format
func (w *WhatsAppProvider) convertToWhatsAppMessage(ctx context.Context, msg msgx.Message) (*whatsappMessage, error) {
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

		// Build components based on template structure
		if len(msg.Content.Template.Parameters) > 0 {
			// Try to get template to determine parameter format
			template, err := w.GetTemplate(ctx, msg.Content.Template.Name, msg.Content.Template.Language)
			if err != nil {
				logx.Error("Failed to get template structure: %v", err)
				// Fallback to named parameter handling
				components, err := w.buildTemplateComponentsForNamed(ctx, msg.Content.Template)
				if err != nil {
					return w.convertToWhatsAppMessageFallback(msg)
				}
				whatsappMsg.Template.Components = components
			} else {
				// Handle based on parameter format
				logx.Debug("Template parameter format: %s", template.ParameterFormat)

				var components []whatsappTemplateComponent
				var componentErr error

				switch template.ParameterFormat {
				case "NAMED":
					// For NAMED templates, use named parameter handling
					components, componentErr = w.buildTemplateComponentsForNamed(ctx, msg.Content.Template)
				case "POSITIONAL", "":
					// For POSITIONAL/numbered templates, use numbered parameter handling
					components, componentErr = w.buildTemplateComponentsForPositional(ctx, msg.Content.Template)
				default:
					// Unknown format, try NAMED first
					logx.Warn("Unknown parameter format '%s', trying NAMED", template.ParameterFormat)
					components, componentErr = w.buildTemplateComponentsForNamed(ctx, msg.Content.Template)
				}

				if componentErr != nil {
					logx.Error("Failed to build template components: %v", componentErr)
					return w.convertToWhatsAppMessageFallback(msg)
				}

				whatsappMsg.Template.Components = components
			}
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	return whatsappMsg, nil
}

// convertToWhatsAppMessageFallback provides a fallback when template processing fails
func (w *WhatsAppProvider) convertToWhatsAppMessageFallback(msg msgx.Message) (*whatsappMessage, error) {
	whatsappMsg := &whatsappMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               w.cleanPhoneNumber(msg.To),
		Type:             "template",
		Template: &whatsappTemplateMessage{
			Name:     msg.Content.Template.Name,
			Language: whatsappLanguage{Code: msg.Content.Template.Language},
		},
	}

	// Simple fallback - assume all parameters go to body
	if len(msg.Content.Template.Parameters) > 0 {
		component := whatsappTemplateComponent{
			Type: "body",
		}

		// Sort parameters by key to ensure consistent ordering
		var sortedKeys []string
		for key := range msg.Content.Template.Parameters {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			component.Parameters = append(component.Parameters, whatsappTemplateParameter{
				Type: "text",
				Text: fmt.Sprintf("%v", msg.Content.Template.Parameters[key]),
			})
		}

		whatsappMsg.Template.Components = []whatsappTemplateComponent{component}
	}

	return whatsappMsg, nil
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

// VerifyWebhook verifies the webhook signature according to WhatsApp Cloud API spec
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

	// Read body once and preserve it
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "read_body")
	}

	// Restore body for subsequent processing
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Extract hex signature (remove "sha256=" prefix if present)
	expectedSigHex := signature
	if strings.HasPrefix(signature, "sha256=") {
		expectedSigHex = signature[7:] // Remove "sha256=" prefix
	}

	// Calculate HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(w.config.WebhookSecret))
	mac.Write(body)
	calculatedSigHex := hex.EncodeToString(mac.Sum(nil))

	// Secure comparison of hex strings
	if !hmac.Equal([]byte(expectedSigHex), []byte(calculatedSigHex)) {
		return msgx.Registry.New(msgx.ErrWebhookVerificationFailed).
			WithDetail("provider", whatsappProvider).
			WithDetail("reason", "Invalid signature").
			WithDetail("received", expectedSigHex).
			WithDetail("calculated", calculatedSigHex).
			WithDetail("body_length", len(body)).
			WithDetail("secret_length", len(w.config.WebhookSecret))
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

func (w *WhatsAppProvider) sendMessage(ctx context.Context, message *whatsappMessage) (*whatsappSendResponse, error) {
	url := fmt.Sprintf("%s/messages", w.baseURL)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "marshal_message")
	}

	logx.Debug("Sending WhatsApp message: %s", string(jsonData))

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
	tsInt, err := strconv.ParseInt(message.Timestamp, 10, 64)
	if err != nil {
		logx.Error("Invalid timestamp: %s", err)
		tsInt = 0
	}
	incomingMsg := &msgx.IncomingMessage{
		ID:        message.ID,
		Provider:  whatsappProvider,
		From:      message.From,
		To:        metadata.PhoneNumberID,
		Timestamp: time.Unix(tsInt, 0),
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

// ========== Typing Indicator Methods ==========

func (w *WhatsAppProvider) SendTypingIndicator(ctx context.Context, to string, isTyping bool) error {
	typingType := "typing_on"
	if !isTyping {
		typingType = "typing_off"
	}

	// Validate phone number format
	cleanedTo := w.cleanPhoneNumber(to)
	if !w.isValidPhoneFormat(cleanedTo) {
		return msgx.Registry.New(msgx.ErrInvalidMessage).
			WithDetail("provider", whatsappProvider).
			WithDetail("phone_number", to).
			WithDetail("cleaned", cleanedTo).
			WithDetail("reason", "Invalid phone number format")
	}

	// Create typing indicator message
	typingMsg := whatsappTypingMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               cleanedTo,
		Type:             typingType,
	}

	// Send the typing indicator
	_, err := w.sendTypingIndicator(ctx, typingMsg)
	if err != nil {
		return msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "send_typing_indicator").
			WithDetail("to", cleanedTo).
			WithDetail("typing_type", typingType)
	}

	return nil
}

// StartTyping sends "typing_on" indicator
func (w *WhatsAppProvider) StartTyping(ctx context.Context, to string) error {
	return w.SendTypingIndicator(ctx, to, true)
}

// StopTyping sends "typing_off" indicator
func (w *WhatsAppProvider) StopTyping(ctx context.Context, to string) error {
	return w.SendTypingIndicator(ctx, to, false)
}

// SendTypingWithDuration sends typing indicator for a specific duration
func (w *WhatsAppProvider) SendTypingWithDuration(ctx context.Context, to string, duration time.Duration) error {
	// Start typing
	if err := w.StartTyping(ctx, to); err != nil {
		return err
	}

	// Wait for the specified duration (max 30 seconds as per WhatsApp limits)
	maxDuration := 30 * time.Second
	if duration > maxDuration {
		duration = maxDuration
	}

	select {
	case <-ctx.Done():
		// Context cancelled, still try to stop typing
		_ = w.StopTyping(context.Background(), to)
		return ctx.Err()
	case <-time.After(duration):
		// Duration elapsed, stop typing
		return w.StopTyping(ctx, to)
	}
}

// sendTypingIndicator handles the actual HTTP request for typing indicators
func (w *WhatsAppProvider) sendTypingIndicator(ctx context.Context, typingMsg whatsappTypingMessage) (*whatsappTypingResponse, error) {
	// Marshal the typing message
	payload, err := json.Marshal(typingMsg)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "marshal_typing_payload")
	}

	// Create the request URL (same endpoint as regular messages)
	url := fmt.Sprintf("%s/messages", w.baseURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "create_typing_request")
	}

	// Set headers (same pattern as regular messages)
	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "http_typing_request")
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, w.handleAPIError(resp)
	}

	// Parse response
	var typingResp whatsappTypingResponse
	if err := json.NewDecoder(resp.Body).Decode(&typingResp); err != nil {
		return nil, msgx.Registry.New(msgx.ErrSendFailed).
			WithCause(err).
			WithDetail("provider", whatsappProvider).
			WithDetail("operation", "decode_typing_response")
	}

	return &typingResp, nil
}

// SendWithTyping sends a message with typing indicator simulation
func (w *WhatsAppProvider) SendWithTyping(ctx context.Context, message msgx.Message, typingDuration time.Duration) (*msgx.Response, error) {
	// Start typing indicator
	if err := w.StartTyping(ctx, message.To); err != nil {
		// Log the error but don't fail the message send
		logx.Error("Failed to send typing indicator before message: %v", err)
	}

	// Wait for typing duration
	if typingDuration > 0 {
		maxDuration := 30 * time.Second
		if typingDuration > maxDuration {
			typingDuration = maxDuration
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(typingDuration):
			// Continue to send message
		}
	}

	// Send the actual message
	response, err := w.Send(ctx, message)

	// Stop typing indicator (fire and forget)
	go func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if stopErr := w.StopTyping(stopCtx, message.To); stopErr != nil {
			logx.Error("Failed to stop typing indicator after message: %v", stopErr)
		}
	}()

	return response, err
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
	Timestamp string                    `json:"timestamp"`
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
	} `json:"referred"`
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
	Org       whatsappContactOrg       `json:"org"`
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

type whatsappTypingMessage struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type,omitempty"`
	To               string `json:"to"`
	Type             string `json:"type"` // "typing_on" or "typing_off"
}

// Response structure for typing indicator (same as regular send response)
type whatsappTypingResponse struct {
	MessagingProduct string                    `json:"messaging_product"`
	Contacts         []whatsappContact         `json:"contacts"`
	Messages         []whatsappMessageResponse `json:"messages"`
}
