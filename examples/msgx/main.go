package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/providers/msgxwhatsapp"
)

func main() {
	// Configuration - load from environment variables
	config := msgxwhatsapp.WhatsAppConfig{
		AccessToken:   getEnv("WHATSAPP_ACCESS_TOKEN", ""),
		PhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
		WebhookSecret: getEnv("WHATSAPP_WEBHOOK_SECRET", ""),
		VerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", ""),
		HTTPTimeout:   30,
	}

	// Validate configuration
	if config.AccessToken == "" || config.PhoneNumberID == "" {
		log.Fatal("WHATSAPP_ACCESS_TOKEN and WHATSAPP_PHONE_NUMBER_ID are required")
	}

	// Create WhatsApp provider (implements msgx.Provider interface)
	whatsappProvider := msgxwhatsapp.NewWhatsAppProvider(config)

	// Create webhook server
	webhookServer := msgx.NewWebhookServer(8080)

	// Setup webhook configuration
	webhookConfig := msgx.WebhookConfig{
		URL:         "https://yourdomain.com/webhook/whatsapp",
		Secret:      config.WebhookSecret,
		VerifyToken: config.VerifyToken,
		Events: []msgx.EventType{
			msgx.EventMessageReceived,
			msgx.EventStatusUpdate,
		},
	}

	// Setup webhook on the provider
	if err := whatsappProvider.SetupWebhook(webhookConfig); err != nil {
		log.Fatalf("Failed to setup webhook: %v", err)
	}

	// Register the provider directly (it implements msgx.Receiver)
	webhookServer.RegisterProvider("/webhook/whatsapp", whatsappProvider)

	// Create message processor
	messageProcessor := NewMessageProcessor(whatsappProvider)

	// Add custom route with message processing
	http.HandleFunc("/webhook/whatsapp", func(w http.ResponseWriter, r *http.Request) {
		handleWhatsAppWebhook(w, r, whatsappProvider, messageProcessor)
	})

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server in goroutine
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Println("Starting WhatsApp webhook server on :8080...")
		log.Println("Webhook endpoint: http://localhost:8080/webhook/whatsapp")
		log.Println("Health check: http://localhost:8080/health")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Test sending a message (optional)
	go func() {
		time.Sleep(3 * time.Second) // Wait for server to start
		testSendMessage(whatsappProvider)
	}()

	// Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

// handleWhatsAppWebhook handles the webhook endpoint
func handleWhatsAppWebhook(w http.ResponseWriter, r *http.Request, provider *msgxwhatsapp.WhatsAppProvider, processor *MessageProcessor) {
	ctx := r.Context()

	// Handle GET requests (webhook verification)
	if r.Method == "GET" {
		handleWebhookVerification(w, r, provider)
		return
	}

	// Handle POST requests (incoming messages/status updates)
	if r.Method == "POST" {
		log.Printf("Received webhook: %s %s", r.Method, r.URL.Path)

		// Use the provider to handle the webhook
		incomingMessage, err := provider.HandleWebhook(ctx, r)
		if err != nil {
			log.Printf("Error handling webhook: %v", err)
			http.Error(w, "Webhook processing failed", http.StatusInternalServerError)
			return
		}

		// Process the message if it exists
		if incomingMessage != nil {
			log.Printf("Processing incoming message: %+v", incomingMessage)

			if err := processor.ProcessMessage(ctx, incomingMessage); err != nil {
				log.Printf("Error processing message: %v", err)
				// Don't return error to WhatsApp - message was received successfully
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// Method not allowed
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleWebhookVerification handles WhatsApp webhook verification challenge
func handleWebhookVerification(w http.ResponseWriter, r *http.Request, provider *msgxwhatsapp.WhatsAppProvider) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	log.Printf("Webhook verification: mode=%s, token=%s, challenge=%s", mode, token, challenge)

	// Verify using the provider's verification method
	if err := provider.VerifyWebhook(r); err != nil {
		log.Printf("Webhook verification failed: %v", err)
		http.Error(w, "Verification failed", http.StatusForbidden)
		return
	}

	if mode == "subscribe" && challenge != "" {
		log.Println("Webhook verified successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	log.Println("Invalid verification request")
	http.Error(w, "Bad request", http.StatusBadRequest)
}

// MessageProcessor handles message processing logic
type MessageProcessor struct {
	provider msgx.Sender
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(provider msgx.Sender) *MessageProcessor {
	return &MessageProcessor{
		provider: provider,
	}
}

// ProcessMessage processes an incoming message
func (mp *MessageProcessor) ProcessMessage(ctx context.Context, message *msgx.IncomingMessage) error {
	log.Printf("Processing message from %s (type: %s)", message.From, message.Type)

	// Handle different message types
	switch message.Type {
	case msgx.MessageTypeText:
		return mp.handleTextMessage(ctx, message)
	case msgx.MessageTypeImage:
		return mp.handleMediaMessage(ctx, message, "image")
	case msgx.MessageTypeDocument:
		return mp.handleMediaMessage(ctx, message, "document")
	case msgx.MessageTypeAudio:
		return mp.handleMediaMessage(ctx, message, "audio")
	case msgx.MessageTypeVideo:
		return mp.handleMediaMessage(ctx, message, "video")
	default:
		return mp.sendReply(ctx, message.From, fmt.Sprintf("Received a %s message. Thanks!", message.Type))
	}
}

// handleTextMessage handles incoming text messages
func (mp *MessageProcessor) handleTextMessage(ctx context.Context, message *msgx.IncomingMessage) error {
	if message.Content.Text == nil {
		return fmt.Errorf("text content is nil")
	}

	text := message.Content.Text.Body
	log.Printf("Received text message: %s", text)

	// Simple command processing
	switch text {
	case "/help":
		return mp.sendReply(ctx, message.From, "Available commands:\n/help - Show this help\n/info - Get bot info\n/echo <text> - Echo your message")
	case "/info":
		return mp.sendReply(ctx, message.From, "WhatsApp Bot v1.0\nPowered by msgx framework")
	default:
		if len(text) > 5 && text[:5] == "/echo" {
			echoText := text[5:]
			if len(echoText) > 0 {
				return mp.sendReply(ctx, message.From, "Echo: "+echoText)
			}
		}
		// Default response
		return mp.sendReply(ctx, message.From, fmt.Sprintf("You said: %s\n\nSend /help for available commands.", text))
	}
}

// handleMediaMessage handles incoming media messages
func (mp *MessageProcessor) handleMediaMessage(ctx context.Context, message *msgx.IncomingMessage, mediaType string) error {
	responseText := fmt.Sprintf("Received a %s!", mediaType)

	if message.Content.Media != nil && message.Content.Media.Caption != "" {
		responseText += fmt.Sprintf("\nCaption: %s", message.Content.Media.Caption)
	}

	return mp.sendReply(ctx, message.From, responseText)
}

// sendReply sends a reply message
func (mp *MessageProcessor) sendReply(ctx context.Context, to, text string) error {
	reply := msgx.Message{
		To:   to,
		Type: msgx.MessageTypeText,
		Content: msgx.Content{
			Text: &msgx.TextContent{
				Body: text,
			},
		},
	}

	response, err := mp.provider.Send(ctx, reply)
	if err != nil {
		log.Printf("Failed to send reply to %s: %v", to, err)
		return err
	}

	log.Printf("Sent reply to %s: %s (message ID: %s)", to, text, response.MessageID)
	return nil
}

// testSendMessage sends a test message (for development)
func testSendMessage(provider *msgxwhatsapp.WhatsAppProvider) {
	testNumber := getEnv("TEST_WHATSAPP_NUMBER", "")
	if testNumber == "" {
		log.Println("TEST_WHATSAPP_NUMBER not set, skipping test message")
		return
	}

	message := msgx.Message{
		To:   testNumber,
		Type: msgx.MessageTypeText,
		Content: msgx.Content{
			Text: &msgx.TextContent{
				Body: "🚀 WhatsApp webhook server is running!\n\nSend me a message to test the bot.",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := provider.Send(ctx, message)
	if err != nil {
		log.Printf("Failed to send test message: %v", err)
		return
	}

	log.Printf("Test message sent successfully to %s (ID: %s)", testNumber, response.MessageID)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

