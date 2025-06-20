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

	// Set up webhook handler (handles both verification and messages)
	http.HandleFunc("/webhook/whatsapp", msgxwhatsapp.WhatsAppWebhookHandler(provider, messageProcessor))

	// Optional: Debug handler
	http.HandleFunc("/webhook/whatsapp/debug", msgxwhatsapp.DebugWhatsAppWebhook())

	// Manual verification endpoint for testing
	http.HandleFunc("/webhook/whatsapp/verify", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Verification request received - Method: %s", r.Method)
		log.Printf("Query parameters: %v", r.URL.Query())

		if r.Method == "GET" {
			mode := r.URL.Query().Get("hub.mode")
			token := r.URL.Query().Get("hub.verify_token")
			challenge := r.URL.Query().Get("hub.challenge")

			log.Printf("Verification details - Mode: %s, Token: %s, Challenge: %s", mode, token, challenge)
			log.Printf("Expected token: %s", config.VerifyToken)

			if mode == "subscribe" && token == config.VerifyToken {
				log.Println("✅ Webhook verification successful!")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}

			log.Println("❌ Webhook verification failed - token mismatch")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden"))
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Root endpoint with instructions
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Webhook Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .code { background: #e8e8e8; padding: 5px; font-family: monospace; }
    </style>
</head>
<body>
    <h1>WhatsApp Webhook Server</h1>
    <p>Server is running and ready to receive webhooks.</p>
    
    <h2>Available Endpoints:</h2>
    
    <div class="endpoint">
        <h3>Main Webhook Endpoint</h3>
        <div class="code">POST/GET /webhook/whatsapp</div>
        <p>Use this URL in your Facebook Developer Console for webhook configuration.</p>
    </div>
    
    <div class="endpoint">
        <h3>Manual Verification Endpoint</h3>
        <div class="code">GET /webhook/whatsapp/verify</div>
        <p>Test webhook verification manually with parameters:</p>
        <ul>
            <li><code>hub.mode=subscribe</code></li>
            <li><code>hub.verify_token=your_verify_token</code></li>
            <li><code>hub.challenge=test_challenge</code></li>
        </ul>
        <p>Example: <a href="/webhook/whatsapp/verify?hub.mode=subscribe&hub.verify_token=your_verify_token&hub.challenge=test123">/webhook/whatsapp/verify?hub.mode=subscribe&hub.verify_token=your_verify_token&hub.challenge=test123</a></p>
    </div>
    
    <div class="endpoint">
        <h3>Debug Endpoint</h3>
        <div class="code">POST/GET /webhook/whatsapp/debug</div>
        <p>Logs all incoming webhook data for debugging purposes.</p>
    </div>
    
    <div class="endpoint">
        <h3>Health Check</h3>
        <div class="code">GET /health</div>
        <p>Simple health check endpoint.</p>
    </div>
    
    <h2>Configuration:</h2>
    <ul>
        <li><strong>Phone Number ID:</strong> ` + config.PhoneNumberID + `</li>
        <li><strong>Verify Token:</strong> ` + config.VerifyToken + `</li>
        <li><strong>Webhook Secret:</strong> ` + (func() string {
			if config.WebhookSecret != "" {
				return "✅ Configured"
			}
			return "❌ Not configured"
		})() + `</li>
    </ul>
    
    <h2>Testing Steps:</h2>
    <ol>
        <li>Test the manual verification endpoint first</li>
        <li>Configure the webhook URL in Facebook Developer Console</li>
        <li>Send a test message to your WhatsApp Business number</li>
        <li>Check the server logs for incoming messages</li>
    </ol>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	log.Println("🚀 WhatsApp Webhook Server starting on :8080")
	log.Printf("📞 Phone Number ID: %s", config.PhoneNumberID)
	log.Printf("🔑 Verify Token: %s", config.VerifyToken)
	log.Printf("🔐 Webhook Secret: %s", func() string {
		if config.WebhookSecret != "" {
			return "✅ Configured"
		}
		return "❌ Not configured"
	}())
	log.Println("")
	log.Println("📍 Available endpoints:")
	log.Println("   Main webhook:        http://localhost:8080/webhook/whatsapp")
	log.Println("   Manual verification: http://localhost:8080/webhook/whatsapp/verify")
	log.Println("   Debug endpoint:      http://localhost:8080/webhook/whatsapp/debug")
	log.Println("   Health check:        http://localhost:8080/health")
	log.Println("   Instructions:        http://localhost:8080/")
	log.Println("")
	log.Println("🔍 To test verification manually, visit:")
	log.Printf("   http://localhost:8080/webhook/whatsapp/verify?hub.mode=subscribe&hub.verify_token=%s&hub.challenge=test123", config.VerifyToken)
	log.Println("")
	log.Println("⚡ Server ready! Waiting for webhooks...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

