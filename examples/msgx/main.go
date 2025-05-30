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

	"github.com/Abraxas-365/craftable/configx"
	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/msgxproviders"
)

func main() {
	// Set environment variables programmatically for demo purposes
	os.Setenv("MSGX_WHATSAPP_PHONE_NUMBER_ID", "your_phone_number_id")
	os.Setenv("MSGX_WHATSAPP_ACCESS_TOKEN", "your_access_token")
	os.Setenv("MSGX_WHATSAPP_WEBHOOK_SECRET", "your_webhook_secret")
	os.Setenv("MSGX_WHATSAPP_VERIFY_TOKEN", "your_verify_token")
	os.Setenv("MSGX_WHATSAPP_API_VERSION", "v18.0")
	os.Setenv("MSGX_WHATSAPP_HTTP_TIMEOUT", "30")
	os.Setenv("MSGX_WEBHOOK_PORT", "8080")
	os.Setenv("MSGX_WEBHOOK_BASE_URL", "https://yourdomain.com")
	os.Setenv("MSGX_TEST_PHONE_NUMBER", "+1234567890")
	os.Setenv("MSGX_DEBUG", "true")

	// Create configuration from environment variables and defaults
	config, err := configx.NewBuilder().
		WithDefaults(map[string]any{
			"whatsapp": map[string]any{
				"api_version":  "v18.0",
				"http_timeout": 30,
			},
			"webhook": map[string]any{
				"port":     8080,
				"base_url": "http://localhost:8080",
			},
			"debug": false,
		}).
		FromEnv("MSGX_").
		RequireEnv(
			"MSGX_WHATSAPP_PHONE_NUMBER_ID",
			"MSGX_WHATSAPP_ACCESS_TOKEN",
		).
		Build()

	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	// Create WhatsApp provider configuration from config
	whatsappConfig := msgxproviders.WhatsAppConfig{
		PhoneNumberID: config.Get("whatsapp.phone.number.id").AsString(),
		AccessToken:   config.Get("whatsapp.access.token").AsString(),
		WebhookSecret: config.Get("whatsapp.webhook.secret").AsString(),
		VerifyToken:   config.Get("whatsapp.verify.token").AsString(),
		APIVersion:    config.Get("whatsapp.api.version").AsString(),
		HTTPTimeout:   config.Get("whatsapp.http.timeout").AsInt(),
	}

	fmt.Println("WhatsApp Provider Configuration:", whatsappConfig)

	// Get webhook configuration
	webhookPort := config.Get("webhook.port").AsInt()
	baseURL := config.Get("webhook.base_url").AsString()
	debug := config.Get("debug").AsBool()

	if debug {
		log.Println("Debug mode enabled")
		log.Printf("WhatsApp Config: Phone Number ID: %s, API Version: %s",
			whatsappConfig.PhoneNumberID, whatsappConfig.APIVersion)
		log.Printf("Webhook Config: Port: %d, Base URL: %s", webhookPort, baseURL)
	}

	// Create WhatsApp provider
	whatsappProvider := msgxproviders.NewWhatsAppProvider(whatsappConfig)

	// Create messaging service and register WhatsApp provider
	service := msgx.NewService().
		RegisterProvider("whatsapp", whatsappProvider, true).
		OnMessage(func(ctx context.Context, message *msgx.IncomingMessage) error {
			return handleIncomingMessage(ctx, message, config)
		})

	// Set up webhooks for receiving messages
	if err := service.SetupWebhooks(webhookPort, baseURL); err != nil {
		log.Printf("Failed to setup webhooks: %v", err)
	}

	// Start webhook server in a goroutine
	go func() {
		log.Printf("Starting webhook server on port %d", webhookPort)
		if err := service.StartWebhookServer(); err != nil && err != http.ErrServerClosed {
			log.Printf("Webhook server error: %v", err)
		}
	}()

	// Add health check endpoint
	setupHealthCheck(service, config)

	// Demonstrate sending different types of messages
	ctx := context.Background()

	testPhoneNumber := config.Get("test.phone_number").AsString()
	if testPhoneNumber == "" {
		log.Println("MSGX_TEST_PHONE_NUMBER not set, skipping message sending examples")
	} else {
		demonstrateMessaging(ctx, service, testPhoneNumber, config)
	}

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Messaging service is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")
}

func demonstrateMessaging(ctx context.Context, service *msgx.Service, phoneNumber string, config configx.Config) {
	debug := config.Get("debug").AsBool()

	if debug {
		log.Printf("Sending example messages to %s", phoneNumber)
	}

	// 1. Send a simple text message
	textMessage := msgx.Message{
		To:   phoneNumber,
		Type: msgx.MessageTypeText,
		Content: msgx.Content{
			Text: &msgx.TextContent{
				Body:       "Hello! This is a test message from the msgx system with configx integration.",
				PreviewURL: true,
			},
		},
		Options: &msgx.MessageOptions{
			Priority: msgx.PriorityNormal,
		},
		Metadata: map[string]string{
			"source":      "example_app",
			"environment": getEnvironment(config),
		},
	}

	response, err := service.Send(ctx, textMessage)
	if err != nil {
		log.Printf("Failed to send text message: %v", err)
	} else {
		log.Printf("Text message sent successfully. ID: %s", response.MessageID)

		// Check message status after a delay
		if debug {
			time.Sleep(2 * time.Second)
			status, err := service.GetStatus(ctx, response.MessageID)
			if err != nil {
				log.Printf("Failed to get message status: %v", err)
			} else {
				log.Printf("Message status: %s", status.Status)
			}
		}
	}

	// 2. Send an image message with dynamic URL from config
	imageURL := config.Get("example.image_url").AsStringDefault("https://picsum.photos/800/600")
	imageMessage := msgx.Message{
		To:   phoneNumber,
		Type: msgx.MessageTypeImage,
		Content: msgx.Content{
			Media: &msgx.MediaContent{
				URL:     imageURL,
				Caption: "Sample image sent via msgx with configx!",
			},
		},
		Metadata: map[string]string{
			"type":       "demo_image",
			"config_url": imageURL,
		},
	}

	response, err = service.Send(ctx, imageMessage)
	if err != nil {
		log.Printf("Failed to send image message: %v", err)
	} else {
		log.Printf("Image message sent successfully. ID: %s", response.MessageID)
	}

	// 3. Send a document message
	documentURL := config.Get("example.document_url").AsStringDefault("https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf")
	documentMessage := msgx.Message{
		To:   phoneNumber,
		Type: msgx.MessageTypeDocument,
		Content: msgx.Content{
			Media: &msgx.MediaContent{
				URL:      documentURL,
				Caption:  "Sample PDF document from configx",
				Filename: "sample.pdf",
			},
		},
	}

	response, err = service.Send(ctx, documentMessage)
	if err != nil {
		log.Printf("Failed to send document message: %v", err)
	} else {
		log.Printf("Document message sent successfully. ID: %s", response.MessageID)
	}

	// 4. Send a template message using config
	templateName := config.Get("whatsapp.template.name").AsStringDefault("hello_world")
	templateLanguage := config.Get("whatsapp.template.language").AsStringDefault("en_US")

	templateMessage := msgx.Message{
		To:   phoneNumber,
		Type: msgx.MessageTypeTemplate,
		Content: msgx.Content{
			Template: &msgx.TemplateContent{
				Name:     templateName,
				Language: templateLanguage,
				Parameters: map[string]any{
					"1": config.Get("app.name").AsStringDefault("msgx Demo"),
				},
			},
		},
	}

	response, err = service.Send(ctx, templateMessage)
	if err != nil {
		log.Printf("Failed to send template message: %v", err)
	} else {
		log.Printf("Template message sent successfully. ID: %s", response.MessageID)
	}

	// 5. Send bulk messages with configurable count
	bulkCount := config.Get("example.bulk_count").AsIntDefault(2)
	bulkMessages := make([]msgx.Message, bulkCount)

	for i := 0; i < bulkCount; i++ {
		bulkMessages[i] = msgx.Message{
			To:   phoneNumber,
			Type: msgx.MessageTypeText,
			Content: msgx.Content{
				Text: &msgx.TextContent{
					Body: fmt.Sprintf("Bulk message %d of %d (via configx)", i+1, bulkCount),
				},
			},
		}
	}

	bulkResponse, err := service.SendBulk(ctx, bulkMessages)
	if err != nil {
		log.Printf("Failed to send bulk messages: %v", err)
	} else {
		log.Printf("Bulk messages sent. Total sent: %d, Total failed: %d",
			bulkResponse.TotalSent, bulkResponse.TotalFailed)
	}

	// 6. Validate phone number
	validation, err := service.ValidateNumber(ctx, phoneNumber)
	if err != nil {
		log.Printf("Failed to validate number: %v", err)
	} else {
		log.Printf("Number validation - Valid: %t, Carrier: %s, Country: %s",
			validation.IsValid, validation.Carrier, validation.Country)
	}
}

// handleIncomingMessage processes incoming messages from webhooks
func handleIncomingMessage(ctx context.Context, message *msgx.IncomingMessage, config configx.Config) error {
	debug := config.Get("debug").AsBool()
	autoReply := config.Get("webhook.auto_reply").AsBoolDefault(true)

	if debug {
		log.Printf("Received message from %s (Provider: %s, Type: %s)",
			message.From, message.Provider, message.Type)
	}

	switch message.Type {
	case msgx.MessageTypeText:
		if message.Content.Text != nil {
			log.Printf("Text content: %s", message.Content.Text.Body)

			// Auto-reply if enabled in config
			if autoReply {
				replyPrefix := config.Get("webhook.reply_prefix").AsStringDefault("Echo")
				log.Printf("Would auto-reply with: %s: %s", replyPrefix, message.Content.Text.Body)
			}
		}

	case msgx.MessageTypeImage:
		if message.Content.Media != nil {
			log.Printf("Received image: %s (Caption: %s)",
				message.Content.Media.URL, message.Content.Media.Caption)
		}

	case msgx.MessageTypeDocument:
		if message.Content.Media != nil {
			log.Printf("Received document: %s (Filename: %s)",
				message.Content.Media.URL, message.Content.Media.Filename)
		}

	case msgx.MessageTypeAudio:
		if message.Content.Media != nil {
			log.Printf("Received audio: %s", message.Content.Media.URL)
		}

	case msgx.MessageTypeVideo:
		if message.Content.Media != nil {
			log.Printf("Received video: %s (Caption: %s)",
				message.Content.Media.URL, message.Content.Media.Caption)
		}

	default:
		if debug {
			log.Printf("Received unsupported message type: %s", message.Type)
		}
	}

	// Handle message context (replies, forwards, etc.)
	if message.Context != nil && debug {
		if message.Context.ReplyToID != "" {
			log.Printf("This is a reply to message: %s", message.Context.ReplyToID)
		}
		if message.Context.IsForwarded {
			log.Printf("This message was forwarded from: %s", message.Context.ForwardedFrom)
		}
	}

	return nil
}

func setupHealthCheck(service *msgx.Service, config configx.Config) {
	healthPath := config.Get("webhook.health_path").AsStringDefault("/health")

	http.HandleFunc(healthPath, func(w http.ResponseWriter, r *http.Request) {
		// Check if the service is healthy
		status := map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "msgx",
			"version":   config.Get("app.version").AsStringDefault("1.0.0"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Simple JSON response
		fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","service":"%s","version":"%s"}`,
			status["status"], status["timestamp"], status["service"], status["version"])
	})

	log.Printf("Health check endpoint available at %s", healthPath)
}

func getEnvironment(config configx.Config) string {
	return config.Get("app.environment").AsStringDefault("development")
}

// Configuration validation helper
func validateConfig(config configx.Config) error {
	required := []string{
		"whatsapp.phone_number_id",
		"whatsapp.access_token",
	}

	for _, key := range required {
		if config.Get(key).AsString() == "" {
			return fmt.Errorf("required configuration key missing: %s", key)
		}
	}

	return nil
}

// Additional configuration helpers for production use
func loadAdditionalConfig() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"name":        "msgx-demo",
			"version":     "1.0.0",
			"environment": "development",
		},
		"whatsapp": map[string]any{
			"template": map[string]any{
				"name":     "hello_world",
				"language": "en_US",
			},
		},
		"webhook": map[string]any{
			"auto_reply":   true,
			"reply_prefix": "Bot",
			"health_path":  "/health",
		},
		"example": map[string]any{
			"bulk_count":   2,
			"image_url":    "https://picsum.photos/800/600",
			"document_url": "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
		},
	}
}
