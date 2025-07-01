package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/providers/msgxwhatsapp"
	"github.com/gorilla/mux"
)

// ServiceConfig holds the complete service configuration
type ServiceConfig struct {
	// Server settings
	Port     string `json:"port"`
	BasePath string `json:"base_path"`

	// WhatsApp API configuration
	WhatsApp WhatsAppConfig `json:"whatsapp"`

	// Webhook settings
	WebhookPath string `json:"webhook_path"`

	// Business logic settings
	AutoReply           bool   `json:"auto_reply"`
	AutoReplyMessage    string `json:"auto_reply_message"`
	LogIncomingMessages bool   `json:"log_incoming_messages"`
	DebugMode           bool   `json:"debug_mode"`
}

type WhatsAppConfig struct {
	AccessToken   string `json:"access_token"`
	PhoneNumberID string `json:"phone_number_id"`
	BusinessID    string `json:"business_id,omitempty"`
	WebhookSecret string `json:"webhook_secret"`
	VerifyToken   string `json:"verify_token"`
	APIVersion    string `json:"api_version,omitempty"`
	HTTPTimeout   int    `json:"http_timeout,omitempty"`
}

// Custom WhatsApp webhook structures that handle string timestamps
type WhatsAppWebhookPayload struct {
	Object       string                 `json:"object"`
	Entry        []WhatsAppWebhookEntry `json:"entry"`
	HubMode      string                 `json:"hub.mode,omitempty"`
	HubChallenge string                 `json:"hub.challenge,omitempty"`
	HubVerify    string                 `json:"hub.verify_token,omitempty"`
}

type WhatsAppWebhookEntry struct {
	ID      string                  `json:"id"`
	Changes []WhatsAppWebhookChange `json:"changes"`
}

type WhatsAppWebhookChange struct {
	Value WhatsAppWebhookValue `json:"value"`
	Field string               `json:"field"`
}

type WhatsAppWebhookValue struct {
	MessagingProduct string                    `json:"messaging_product"`
	Metadata         WhatsAppMetadata          `json:"metadata"`
	Contacts         []WhatsAppWebhookContact  `json:"contacts,omitempty"`
	Messages         []WhatsAppIncomingMessage `json:"messages,omitempty"`
	Statuses         []WhatsAppStatusUpdate    `json:"statuses,omitempty"`
}

type WhatsAppMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WhatsAppWebhookContact struct {
	Profile WhatsAppProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type WhatsAppProfile struct {
	Name string `json:"name"`
}

// Fixed incoming message structure with string timestamp
type WhatsAppIncomingMessage struct {
	From      string                   `json:"from"`
	ID        string                   `json:"id"`
	Timestamp string                   `json:"timestamp"` // Changed to string to match WhatsApp API
	Type      string                   `json:"type"`
	Text      *WhatsAppTextMessage     `json:"text,omitempty"`
	Image     *WhatsAppMediaMessage    `json:"image,omitempty"`
	Document  *WhatsAppMediaMessage    `json:"document,omitempty"`
	Audio     *WhatsAppMediaMessage    `json:"audio,omitempty"`
	Video     *WhatsAppMediaMessage    `json:"video,omitempty"`
	Voice     *WhatsAppMediaMessage    `json:"voice,omitempty"`
	Sticker   *WhatsAppMediaMessage    `json:"sticker,omitempty"`
	Location  *WhatsAppLocationMessage `json:"location,omitempty"`
	Contacts  []WhatsAppContactMessage `json:"contacts,omitempty"`
}

type WhatsAppTextMessage struct {
	Body string `json:"body"`
}

type WhatsAppMediaMessage struct {
	ID       string `json:"id,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type WhatsAppLocationMessage struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type WhatsAppContactMessage struct {
	Name   WhatsAppContactName    `json:"name"`
	Phones []WhatsAppContactPhone `json:"phones,omitempty"`
	Emails []WhatsAppContactEmail `json:"emails,omitempty"`
}

type WhatsAppContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
}

type WhatsAppContactPhone struct {
	Phone string `json:"phone"`
	Type  string `json:"type,omitempty"`
}

type WhatsAppContactEmail struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"`
}

type WhatsAppStatusUpdate struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	Conversation struct {
		ID     string `json:"id"`
		Origin struct {
			Type string `json:"type"`
		} `json:"origin"`
	} `json:"conversation,omitempty"`
	Pricing struct {
		Billable     bool   `json:"billable"`
		PricingModel string `json:"pricing_model"`
		Category     string `json:"category"`
	} `json:"pricing,omitempty"`
}

// WhatsAppService provides a complete WhatsApp messaging service
type WhatsAppService struct {
	config   ServiceConfig
	provider *msgxwhatsapp.WhatsAppProvider
	server   *http.Server
	router   *mux.Router

	// Message handling
	messageHandlers []MessageHandler
	mu              sync.RWMutex

	// Metrics and monitoring
	stats ServiceStats
}

// ServiceStats tracks service metrics
type ServiceStats struct {
	MessagesReceived  int64     `json:"messages_received"`
	MessagesSent      int64     `json:"messages_sent"`
	WebhookErrors     int64     `json:"webhook_errors"`
	VerificationCalls int64     `json:"verification_calls"`
	LastMessage       time.Time `json:"last_message"`
	StartTime         time.Time `json:"start_time"`
}

// MessageHandler defines the interface for handling incoming messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, message *msgx.IncomingMessage) error
	Priority() int // Lower numbers = higher priority
}

// SendMessageRequest represents a message sending request
type SendMessageRequest struct {
	To      string         `json:"to"`
	Type    string         `json:"type"`
	Content MessageContent `json:"content"`
}

type MessageContent struct {
	Text     *TextContent     `json:"text,omitempty"`
	Media    *MediaContent    `json:"media,omitempty"`
	Template *TemplateContent `json:"template,omitempty"`
}

type TextContent struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type MediaContent struct {
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type TemplateContent struct {
	Name       string                 `json:"name"`
	Language   string                 `json:"language"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() ServiceConfig {
	return ServiceConfig{
		Port:     getEnv("PORT", "8080"),
		BasePath: getEnv("BASE_PATH", "/api/v1"),
		WhatsApp: WhatsAppConfig{
			AccessToken:   mustGetEnv("WHATSAPP_ACCESS_TOKEN"),
			PhoneNumberID: mustGetEnv("WHATSAPP_PHONE_NUMBER_ID"),
			BusinessID:    getEnv("WHATSAPP_BUSINESS_ID", ""),
			WebhookSecret: mustGetEnv("WHATSAPP_WEBHOOK_SECRET"),
			VerifyToken:   mustGetEnv("WHATSAPP_VERIFY_TOKEN"),
			APIVersion:    getEnv("WHATSAPP_API_VERSION", "v21.0"),
			HTTPTimeout:   getEnvAsInt("WHATSAPP_HTTP_TIMEOUT", 30),
		},
		WebhookPath:         getEnv("WEBHOOK_PATH", "/webhook"),
		AutoReply:           getEnvAsBool("AUTO_REPLY", false),
		AutoReplyMessage:    getEnv("AUTO_REPLY_MESSAGE", ""),
		LogIncomingMessages: getEnvAsBool("LOG_INCOMING_MESSAGES", true),
		DebugMode:           getEnvAsBool("DEBUG_MODE", false),
	}
}

// NewWhatsAppService creates a new WhatsApp service instance
func NewWhatsAppService(config ServiceConfig) (*WhatsAppService, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create WhatsApp provider
	whatsappConfig := msgxwhatsapp.WhatsAppConfig{
		AccessToken:       config.WhatsApp.AccessToken,
		PhoneNumberID:     config.WhatsApp.PhoneNumberID,
		BusinessAccountID: config.WhatsApp.BusinessID,
		WebhookSecret:     config.WhatsApp.WebhookSecret,
		VerifyToken:       config.WhatsApp.VerifyToken,
		APIVersion:        config.WhatsApp.APIVersion,
		HTTPTimeout:       config.WhatsApp.HTTPTimeout,
	}

	provider := msgxwhatsapp.NewWhatsAppProvider(whatsappConfig)

	// Set up webhook configuration
	webhookConfig := msgx.WebhookConfig{
		Secret:      config.WhatsApp.WebhookSecret,
		VerifyToken: config.WhatsApp.VerifyToken,
	}
	if err := provider.SetupWebhook(webhookConfig); err != nil {
		return nil, fmt.Errorf("failed to setup webhook: %w", err)
	}

	// Create router
	router := mux.NewRouter()

	service := &WhatsAppService{
		config:   config,
		provider: provider,
		router:   router,
		stats: ServiceStats{
			StartTime: time.Now(),
		},
	}

	// Setup routes
	service.setupRoutes()

	return service, nil
}

// setupRoutes configures all HTTP routes
func (s *WhatsAppService) setupRoutes() {
	// Apply base path if configured
	var apiRouter *mux.Router
	if s.config.BasePath != "" {
		apiRouter = s.router.PathPrefix(s.config.BasePath).Subrouter()
	} else {
		apiRouter = s.router
	}

	// Webhook endpoint (handles both GET verification and POST messages)
	webhookPath := s.config.WebhookPath
	if webhookPath == "" {
		webhookPath = "/webhook"
	}
	apiRouter.HandleFunc(webhookPath, s.handleWebhook).Methods("GET", "POST")

	// API endpoints
	apiRouter.HandleFunc("/send", s.handleSendMessage).Methods("POST")
	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/stats", s.handleStats).Methods("GET")

	// Add middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.recoveryMiddleware)
}

// Start starts the WhatsApp service
func (s *WhatsAppService) Start() error {
	// Create HTTP server
	s.server = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("🚀 WhatsApp Service starting on port %s", s.config.Port)
	log.Printf("📱 Phone Number ID: %s", s.config.WhatsApp.PhoneNumberID)
	log.Printf("🔗 Webhook endpoint: %s%s", s.config.BasePath, s.config.WebhookPath)
	log.Printf("🔐 Webhook Secret: %s...", maskSecret(s.config.WhatsApp.WebhookSecret))
	log.Printf("🎫 Verify Token: %s...", maskSecret(s.config.WhatsApp.VerifyToken))
	log.Printf("🤖 Sofia UTEC Assistant ready!")

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	return s.Stop()
}

// Stop gracefully stops the service
func (s *WhatsAppService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
			return err
		}
	}

	log.Println("✅ Server exited")
	return nil
}

// AddMessageHandler adds a message handler with priority
func (s *WhatsAppService) AddMessageHandler(handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Insert handler in priority order (lower number = higher priority)
	inserted := false
	for i, h := range s.messageHandlers {
		if handler.Priority() < h.Priority() {
			s.messageHandlers = append(s.messageHandlers[:i],
				append([]MessageHandler{handler}, s.messageHandlers[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		s.messageHandlers = append(s.messageHandlers, handler)
	}
}

// handleWebhook handles both GET (verification) and POST (messages) webhook requests
func (s *WhatsAppService) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.handleWebhookVerification(w, r)
		return
	}

	if r.Method == "POST" {
		s.handleIncomingMessage(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleWebhookVerification handles WhatsApp webhook verification challenge
func (s *WhatsAppService) handleWebhookVerification(w http.ResponseWriter, r *http.Request) {
	s.stats.VerificationCalls++

	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	log.Printf("🔍 Webhook verification: mode=%s, token=%s, challenge=%s", mode, token, challenge)

	if mode == "subscribe" && token == s.config.WhatsApp.VerifyToken {
		log.Println("✅ Webhook verification successful")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	log.Printf("❌ Webhook verification failed: invalid token. Expected: %s..., Got: %s",
		maskSecret(s.config.WhatsApp.VerifyToken), token)
	http.Error(w, "Verification failed", http.StatusForbidden)
}

// handleIncomingMessage processes incoming webhook messages with custom parsing
func (s *WhatsAppService) handleIncomingMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Custom webhook processing to handle timestamp issue
	incomingMessage, err := s.parseWhatsAppWebhook(r)
	if err != nil {
		s.stats.WebhookErrors++
		log.Printf("❌ Webhook processing failed: %v", err)
		// Still return 200 to WhatsApp to avoid retries
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// Process the message if it exists
	if incomingMessage != nil {
		s.stats.MessagesReceived++
		s.stats.LastMessage = time.Now()

		if s.config.LogIncomingMessages {
			log.Printf("📥 Incoming message: From=%s, Type=%s, ID=%s, Content=%s",
				incomingMessage.From, incomingMessage.Type, incomingMessage.ID,
				func() string {
					if incomingMessage.Content.Text != nil {
						return incomingMessage.Content.Text.Body
					}
					return fmt.Sprintf("[%s]", incomingMessage.Type)
				}())
		}

		// Process through handlers
		if err := s.processMessage(ctx, incomingMessage); err != nil {
			log.Printf("❌ Message processing failed: %v", err)
		}

		// Auto-reply if enabled (but Sofia should handle this)
		if s.config.AutoReply && s.config.AutoReplyMessage != "" {
			if err := s.sendAutoReply(ctx, incomingMessage); err != nil {
				log.Printf("❌ Auto-reply failed: %v", err)
			}
		}
	}

	// Always respond with 200 to WhatsApp
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// parseWhatsAppWebhook manually parses WhatsApp webhook with proper timestamp handling
func (s *WhatsAppService) parseWhatsAppWebhook(r *http.Request) (*msgx.IncomingMessage, error) {
	// First verify signature using the msgx provider's logic
	if err := s.provider.VerifyWebhook(r); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Parse with our custom structures
	var webhook WhatsAppWebhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
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
				return s.convertWhatsAppMessage(message, change.Value.Metadata)
			}

			// Handle status updates (optional)
			for _, status := range change.Value.Statuses {
				log.Printf("📊 Status update: %s -> %s", status.ID, status.Status)
			}
		}
	}

	return nil, nil
}

// convertWhatsAppMessage converts WhatsApp message to msgx format with proper timestamp handling
func (s *WhatsAppService) convertWhatsAppMessage(message WhatsAppIncomingMessage, metadata WhatsAppMetadata) (*msgx.IncomingMessage, error) {
	// Convert string timestamp to time.Time
	timestampInt, err := strconv.ParseInt(message.Timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %w", err)
	}

	incomingMsg := &msgx.IncomingMessage{
		ID:        message.ID,
		Provider:  "whatsapp",
		From:      message.From,
		To:        metadata.PhoneNumberID,
		Timestamp: time.Unix(timestampInt, 0),
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

	case "document":
		incomingMsg.Type = msgx.MessageTypeDocument
		incomingMsg.Content.Media = &msgx.IncomingMediaContent{
			Caption:  message.Document.Caption,
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

	default:
		// Handle unknown types as text
		incomingMsg.Type = msgx.MessageTypeText
		incomingMsg.Content.Text = &msgx.IncomingTextContent{
			Body: fmt.Sprintf("Received %s message", message.Type),
		}
	}

	return incomingMsg, nil
}

// processMessage processes incoming messages through registered handlers
func (s *WhatsAppService) processMessage(ctx context.Context, message *msgx.IncomingMessage) error {
	s.mu.RLock()
	handlers := make([]MessageHandler, len(s.messageHandlers))
	copy(handlers, s.messageHandlers)
	s.mu.RUnlock()

	// Process through handlers in priority order
	for _, handler := range handlers {
		if err := handler.HandleMessage(ctx, message); err != nil {
			log.Printf("⚠️ Handler error: %v", err)
			continue
		}
	}

	return nil
}

// sendAutoReply sends an automatic reply
func (s *WhatsAppService) sendAutoReply(ctx context.Context, incomingMessage *msgx.IncomingMessage) error {
	message := msgx.Message{
		To:   incomingMessage.From,
		Type: msgx.MessageTypeText,
		Content: msgx.Content{ // Fixed: Use MessageContent instead of Content
			Text: &msgx.TextContent{
				Body: s.config.AutoReplyMessage,
			},
		},
	}

	_, err := s.provider.Send(ctx, message)
	if err == nil {
		s.stats.MessagesSent++
		log.Printf("🤖 Auto-reply sent to %s", incomingMessage.From)
	}
	return err
}

// handleSendMessage handles API requests to send messages
func (s *WhatsAppService) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert to msgx.Message
	message, err := s.convertToMsgxMessage(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid message: %v", err), http.StatusBadRequest)
		return
	}

	// Send message
	response, err := s.provider.Send(ctx, *message)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		http.Error(w, fmt.Sprintf("Failed to send message: %v", err), http.StatusInternalServerError)
		return
	}

	s.stats.MessagesSent++
	log.Printf("📤 Message sent to %s, ID: %s", req.To, response.MessageID)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message_id": response.MessageID,
		"to":         req.To,
	})
}

// convertToMsgxMessage converts API request to msgx.Message
func (s *WhatsAppService) convertToMsgxMessage(req SendMessageRequest) (*msgx.Message, error) {
	message := &msgx.Message{
		To: req.To,
	}

	switch strings.ToLower(req.Type) {
	case "text":
		if req.Content.Text == nil {
			return nil, fmt.Errorf("text content required for text messages")
		}
		message.Type = msgx.MessageTypeText
		message.Content.Text = &msgx.TextContent{
			Body:       req.Content.Text.Body,
			PreviewURL: req.Content.Text.PreviewURL,
		}

	case "image":
		if req.Content.Media == nil {
			return nil, fmt.Errorf("media content required for image messages")
		}
		message.Type = msgx.MessageTypeImage
		message.Content.Media = &msgx.MediaContent{
			URL:     req.Content.Media.URL,
			Caption: req.Content.Media.Caption,
		}

	case "document":
		if req.Content.Media == nil {
			return nil, fmt.Errorf("media content required for document messages")
		}
		message.Type = msgx.MessageTypeDocument
		message.Content.Media = &msgx.MediaContent{
			URL:      req.Content.Media.URL,
			Caption:  req.Content.Media.Caption,
			Filename: req.Content.Media.Filename,
		}

	case "template":
		if req.Content.Template == nil {
			return nil, fmt.Errorf("template content required for template messages")
		}
		message.Type = msgx.MessageTypeTemplate
		message.Content.Template = &msgx.TemplateContent{
			Name:       req.Content.Template.Name,
			Language:   req.Content.Template.Language,
			Parameters: req.Content.Template.Parameters,
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", req.Type)
	}

	return message, nil
}

// handleHealth returns service health status
func (s *WhatsAppService) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(s.stats.StartTime).String(),
		"provider":  "whatsapp",
		"phone":     getEnv("WHATSAPP_DISPLAY_PHONE", "+1 929 708 5087"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleStats returns service statistics
func (s *WhatsAppService) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.stats)
}

// Middleware functions

func (s *WhatsAppService) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *WhatsAppService) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("💥 Panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Utility functions

func validateConfig(config ServiceConfig) error {
	if config.Port == "" {
		return fmt.Errorf("port is required")
	}
	if config.WhatsApp.AccessToken == "" {
		return fmt.Errorf("whatsapp access token is required")
	}
	if config.WhatsApp.PhoneNumberID == "" {
		return fmt.Errorf("whatsapp phone number ID is required")
	}
	if config.WhatsApp.WebhookSecret == "" {
		return fmt.Errorf("webhook secret is required")
	}
	if config.WhatsApp.VerifyToken == "" {
		return fmt.Errorf("verify token is required")
	}
	return nil
}

// Sofia UTEC Message Handler - Responds in Spanish
type SofiaMessageHandler struct {
	service *WhatsAppService
}

func (h *SofiaMessageHandler) HandleMessage(ctx context.Context, message *msgx.IncomingMessage) error {
	if message.Content.Text == nil {
		log.Printf("🤖 Sofia: Received non-text message of type %s from %s", message.Type, message.From)
		return h.sendResponse(ctx, message.From, "¡Hola! Solo puedo procesar mensajes de texto por ahora. ¿Podrías escribir tu consulta?")
	}

	body := strings.ToLower(strings.TrimSpace(message.Content.Text.Body))

	log.Printf("🤖 Sofia: Processing message '%s' from %s", body, message.From)

	// Sofia's intelligent responses for UTEC
	var response string
	switch {
	case strings.Contains(body, "hola") || strings.Contains(body, "hello") || strings.Contains(body, "hi"):
		response = "¡Hola! 👋 Soy Sofia, tu asistente virtual de UTEC.\n\n¿En qué puedo ayudarte hoy?\n\n📚 Puedo informarte sobre:\n• Admisiones y requisitos\n• Carreras disponibles\n• Becas y financiamiento\n• Campus y ubicación\n• Contactos importantes"

	case strings.Contains(body, "admision") || strings.Contains(body, "admission") || strings.Contains(body, "postular") || strings.Contains(body, "inscripcion"):
		response = "📋 **ADMISIONES UTEC 2025**\n\n🔹 **Proceso de admisión:**\n• Examen de admisión presencial\n• Evaluación de expediente académico\n• Entrevista personal\n\n🔹 **Fechas importantes:**\n• Inscripciones: Enero - Febrero\n• Exámenes: Marzo\n\n📞 **Más información:**\n• Web: www.utec.edu.pe/admision\n• Teléfono: (01) 230-5000\n• Email: admision@utec.edu.pe"

	case strings.Contains(body, "carrera") || strings.Contains(body, "program") || strings.Contains(body, "estudiar") || strings.Contains(body, "ingenieria"):
		response = "🎓 **CARRERAS EN UTEC**\n\n🔬 **Ingeniería:**\n• Ingeniería Civil\n• Ingeniería Eléctrica\n• Ingeniería Mecánica\n• Ingeniería Industrial\n• Ingeniería de Minas\n\n💻 **Tecnología:**\n• Ciencias de la Computación\n• Ingeniería de Sistemas\n• Data Science\n\n📊 **Gestión:**\n• Administración y Negocios Digitales\n\n¿Te interesa alguna carrera en particular? ¡Puedo darte más detalles!"

	case strings.Contains(body, "becas") || strings.Contains(body, "scholarship") || strings.Contains(body, "beca") || strings.Contains(body, "financiamiento"):
		response = "💰 **BECAS Y FINANCIAMIENTO UTEC**\n\n🏆 **Becas disponibles:**\n• Beca de Excelencia Académica (50-100%)\n• Beca Socioeconómica (25-75%)\n• Beca por Ubicación Geográfica\n• Beca Deportiva\n• Beca Talento\n\n📋 **Requisitos generales:**\n• Promedio mínimo 15/20\n• Situación socioeconómica (algunas becas)\n• Carta de motivación\n\n📧 **Consultas:** becas@utec.edu.pe"

	case strings.Contains(body, "campus") || strings.Contains(body, "ubicacion") || strings.Contains(body, "direccion") || strings.Contains(body, "donde"):
		response = "📍 **CAMPUS UTEC**\n\n🏢 **Ubicación:**\nJr. Medrano Silva 165, Barranco\nLima, Perú\n\n🚌 **Cómo llegar:**\n• Metropolitano: Estación Bulevar\n• Buses: Líneas que van a Barranco\n• Taxi/Uber: Referencias Puente Barranco\n\n🕒 **Horarios:**\n• Lunes a Viernes: 7:00 AM - 10:00 PM\n• Sábados: 8:00 AM - 6:00 PM"

	case strings.Contains(body, "contacto") || strings.Contains(body, "telefono") || strings.Contains(body, "email") || strings.Contains(body, "comunicar"):
		response = "📞 **CONTACTOS UTEC**\n\n🏢 **Oficina Principal:**\n• Teléfono: (01) 230-5000\n• Email: info@utec.edu.pe\n\n📚 **Admisiones:**\n• Email: admision@utec.edu.pe\n• WhatsApp: +51 929 708 5087\n\n💰 **Becas:**\n• Email: becas@utec.edu.pe\n\n🌐 **Web:** www.utec.edu.pe\n📱 **Redes:** @utec.pe"

	case strings.Contains(body, "costo") || strings.Contains(body, "precio") || strings.Contains(body, "pension") || strings.Contains(body, "mensualidad"):
		response = "💳 **COSTOS ACADÉMICOS**\n\nLos costos varían según:\n• Carrera elegida\n• Escala socioeconómica\n• Becas aplicables\n\n📊 **Rango aproximado:**\n• Desde S/ 2,500 hasta S/ 4,500 mensuales\n\n💡 **Recomendación:**\nContacta directamente para una cotización personalizada considerando tu situación:\n📞 (01) 230-5000\n📧 admision@utec.edu.pe"

	case strings.Contains(body, "gracias") || strings.Contains(body, "thank"):
		response = "¡De nada! 😊\n\nEstoy aquí para ayudarte con cualquier consulta sobre UTEC.\n\n¿Hay algo más en lo que pueda asistirte?\n\n💡 También puedes contactar directamente:\n📞 (01) 230-5000\n🌐 www.utec.edu.pe"

	case strings.Contains(body, "horario") || strings.Contains(body, "atencion") || strings.Contains(body, "cuando"):
		response = "🕒 **HORARIOS DE ATENCIÓN**\n\n🏢 **Campus:**\n• Lunes a Viernes: 7:00 AM - 10:00 PM\n• Sábados: 8:00 AM - 6:00 PM\n• Domingos: Cerrado\n\n📞 **Oficinas administrativas:**\n• Lunes a Viernes: 8:00 AM - 6:00 PM\n• Sábados: 8:00 AM - 1:00 PM\n\n🤖 **Sofia (yo):** ¡24/7 a tu servicio!"

	default:
		response = "🤔 Entiendo que necesitas información sobre UTEC.\n\n📚 **Puedo ayudarte con:**\n• ✅ Admisiones y proceso de postulación\n• ✅ Carreras y programas académicos\n• ✅ Becas y financiamiento\n• ✅ Campus y ubicación\n• ✅ Contactos y horarios\n• ✅ Costos académicos\n\n💬 **Escribe algo como:**\n• \"Quiero información sobre admisiones\"\n• \"¿Qué carreras tienen?\"\n• \"¿Hay becas disponibles?\"\n\n¿Sobre qué te gustaría saber más? 😊"
	}

	return h.sendResponse(ctx, message.From, response)
}

func (h *SofiaMessageHandler) sendResponse(ctx context.Context, to, text string) error {
	message := msgx.Message{
		To:   to,
		Type: msgx.MessageTypeText,
		Content: msgx.Content{ // Fixed: Use MessageContent instead of Content
			Text: &msgx.TextContent{
				Body: text,
			},
		},
	}

	_, err := h.service.provider.Send(ctx, message)
	if err == nil {
		h.service.stats.MessagesSent++
		log.Printf("🤖 Sofia responded to %s", to)
	} else {
		log.Printf("❌ Sofia failed to respond: %v", err)
	}
	return err
}

func (h *SofiaMessageHandler) Priority() int {
	return 1 // High priority
}

// Environment variable helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + "..." + strings.Repeat("*", 4)
}

// Main function
func main() {
	log.Println("🎓 SOFIA UTEC - Starting WhatsApp Assistant")

	// Load configuration from environment variables
	config := loadConfigFromEnv()

	log.Printf("📱 Phone Number ID: %s", config.WhatsApp.PhoneNumberID)
	log.Printf("🔐 Environment: %s", getEnv("ENVIRONMENT", "development"))

	// Create service
	service, err := NewWhatsAppService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Add Sofia message handler
	service.AddMessageHandler(&SofiaMessageHandler{service: service})

	log.Println("🤖 Sofia UTEC Assistant is ready!")
	log.Printf("💬 Webhook URL: https://your-domain.com%s%s", config.BasePath, config.WebhookPath)

	// Start service
	if err := service.Start(); err != nil {
		log.Fatalf("Service failed: %v", err)
	}
}
