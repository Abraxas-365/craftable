package main

import (
	"log"
	"net/http"

	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/providers/msgxwhatsapp"
)

func main() {
	// Initialize WhatsApp provider
	config := msgxwhatsapp.WhatsAppConfig{
		PhoneNumberID: "your_phone_number_id",
		AccessToken:   "your_access_token",
		WebhookSecret: "your_webhook_secret",
		VerifyToken:   "your_verify_token",
	}

	provider := msgxwhatsapp.NewWhatsAppProvider(config)

	// Message processor function
	messageProcessor := func(msg *msgx.IncomingMessage) error {
		log.Printf("Received message from %s: %s", msg.From, msg.Type)

		if msg.Content.Text != nil {
			log.Printf("Text: %s", msg.Content.Text.Body)
		}

		// Add your message processing logic here
		return nil
	}

	// Set up webhook handler
	http.HandleFunc("/webhook/whatsapp", msgxwhatsapp.WhatsAppWebhookHandler(provider, messageProcessor))

	// Optional: Debug handler
	http.HandleFunc("/webhook/whatsapp/debug", msgxwhatsapp.DebugWhatsAppWebhook())

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

