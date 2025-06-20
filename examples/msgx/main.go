package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/providers/msgxwhatsapp"
)

// TestSignatureGeneration helps debug signature issues
func TestSignatureGeneration(secret, payload string) {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	log.Printf("=== Signature Test ===")
	log.Printf("Secret: '%s'", secret)
	log.Printf("Payload: '%s'", payload)
	log.Printf("Generated signature: %s", signature)
	log.Printf("Full header: sha256=%s", signature)
	log.Printf("========================")
}

// VerifyWebhookHandler creates a handler to test webhook verification
func VerifyWebhookHandler(config msgxwhatsapp.WhatsAppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Test verification challenge
			mode := r.URL.Query().Get("hub.mode")
			token := r.URL.Query().Get("hub.verify_token")
			challenge := r.URL.Query().Get("hub.challenge")

			log.Printf("Verification request:")
			log.Printf("  Mode: %s", mode)
			log.Printf("  Token: %s", token)
			log.Printf("  Challenge: %s", challenge)
			log.Printf("  Expected token: %s", config.VerifyToken)

			if mode == "subscribe" && token == config.VerifyToken {
				log.Printf("✅ Verification successful!")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}

			log.Printf("❌ Verification failed!")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Verification failed"))

		case "POST":
			// Test signature verification
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read body", http.StatusBadRequest)
				return
			}

			signature := r.Header.Get("X-Hub-Signature-256")
			log.Printf("Signature verification test:")
			log.Printf("  Received signature: %s", signature)
			log.Printf("  Body length: %d bytes", len(body))
			log.Printf("  Body: %s", string(body))

			if config.WebhookSecret == "" {
				log.Printf("⚠️  No webhook secret configured - skipping verification")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK - No secret configured"))
				return
			}

			if signature == "" {
				log.Printf("❌ No signature header found")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("No signature header"))
				return
			}

			// Remove "sha256=" prefix
			signature = strings.TrimPrefix(signature, "sha256=")

			// Calculate expected signature
			mac := hmac.New(sha256.New, []byte(config.WebhookSecret))
			mac.Write(body)
			expectedSignature := hex.EncodeToString(mac.Sum(nil))

			log.Printf("  Expected signature: %s", expectedSignature)
			log.Printf("  Webhook secret: %s", config.WebhookSecret)

			if hmac.Equal([]byte(signature), []byte(expectedSignature)) {
				log.Printf("✅ Signature verification successful!")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Signature valid"))
			} else {
				log.Printf("❌ Signature verification failed!")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Invalid signature"))
			}

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// DebugWhatsAppWebhookWithSignature creates a debug handler with signature verification
func DebugWhatsAppWebhookWithSignature(webhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("=== WhatsApp Webhook Debug ===")
		log.Printf("Method: %s", r.Method)
		log.Printf("URL: %s", r.URL.String())
		log.Printf("Headers:")
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("  %s: %s", name, value)
			}
		}

		if r.Method == "GET" {
			// Handle verification challenge
			query := r.URL.Query()
			mode := query.Get("hub.mode")
			token := query.Get("hub.verify_token")
			challenge := query.Get("hub.challenge")

			log.Printf("GET Parameters:")
			log.Printf("  hub.mode: %s", mode)
			log.Printf("  hub.verify_token: %s", token)
			log.Printf("  hub.challenge: %s", challenge)

			if challenge != "" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Debug GET OK"))
			}
			return
		}

		if r.Method == "POST" {
			// Read body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			log.Printf("Raw body (%d bytes): %s", len(body), string(body))

			// Pretty print JSON if possible
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
				log.Printf("Pretty JSON:\n%s", prettyJSON.String())
			}

			// Test signature verification if secret is provided
			if webhookSecret != "" {
				signature := r.Header.Get("X-Hub-Signature-256")
				log.Printf("Signature verification:")
				log.Printf("  Received: %s", signature)

				if signature != "" {
					// Remove "sha256=" prefix
					signature = strings.TrimPrefix(signature, "sha256=")

					// Calculate expected signature
					mac := hmac.New(sha256.New, []byte(webhookSecret))
					mac.Write(body)
					expectedSignature := hex.EncodeToString(mac.Sum(nil))

					log.Printf("  Expected: %s", expectedSignature)
					log.Printf("  Secret: %s", webhookSecret)

					if hmac.Equal([]byte(signature), []byte(expectedSignature)) {
						log.Printf("  Result: ✅ VALID")
					} else {
						log.Printf("  Result: ❌ INVALID")
					}
				} else {
					log.Printf("  No signature header found")
				}
			}

			// Parse webhook structure
			var webhook map[string]interface{}
			if err := json.Unmarshal(body, &webhook); err == nil {
				log.Printf("Webhook structure:")
				logWebhookStructure(webhook, "  ")
			}

			log.Printf("=== End Debug ===")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DEBUG_OK"))
		}
	}
}

// logWebhookStructure recursively logs the webhook structure
func logWebhookStructure(data interface{}, indent string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			log.Printf("%s%s:", indent, key)
			logWebhookStructure(value, indent+"  ")
		}
	case []interface{}:
		log.Printf("%s[array with %d items]", indent, len(v))
		for i, item := range v {
			log.Printf("%s[%d]:", indent, i)
			logWebhookStructure(item, indent+"  ")
		}
	case string:
		if len(v) > 100 {
			log.Printf("%s\"%s...\" (truncated, full length: %d)", indent, v[:100], len(v))
		} else {
			log.Printf("%s\"%s\"", indent, v)
		}
	case float64:
		log.Printf("%s%g", indent, v)
	case bool:
		log.Printf("%s%t", indent, v)
	case nil:
		log.Printf("%snull", indent)
	default:
		log.Printf("%s%v (type: %T)", indent, v, v)
	}
}

func main() {
	// Initialize WhatsApp provider
	config := msgxwhatsapp.WhatsAppConfig{
		PhoneNumberID: "your_phone_number_id",
		AccessToken:   "your_access_token",
		WebhookSecret: "your_webhook_secret", // Set to "" to disable verification temporarily
		VerifyToken:   "your_verify_token",
	}

	// Test signature generation with sample data
	TestSignatureGeneration(config.WebhookSecret, `{"test":"payload"}`)

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

	// Enhanced debug handler
	http.HandleFunc("/webhook/whatsapp/debug", DebugWhatsAppWebhookWithSignature(config.WebhookSecret))

	// Set up webhook handler
	http.HandleFunc("/webhook/whatsapp", msgxwhatsapp.WhatsAppWebhookHandler(provider, messageProcessor))

	// Verification endpoint for testing
	http.HandleFunc("/webhook/whatsapp/verify", VerifyWebhookHandler(config))

	// Simple test endpoint to verify signature manually
	http.HandleFunc("/test-signature", func(w http.ResponseWriter, r *http.Request) {
		secret := r.URL.Query().Get("secret")
		payload := r.URL.Query().Get("payload")

		if secret == "" || payload == "" {
			http.Error(w, "Missing secret or payload query params", http.StatusBadRequest)
			return
		}

		TestSignatureGeneration(secret, payload)
		w.Write([]byte("Check logs for signature"))
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// No-verification debug endpoint
	http.HandleFunc("/webhook/whatsapp/no-verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			challenge := r.URL.Query().Get("hub.challenge")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}

		if r.Method == "POST" {
			// Create a temporary provider with no webhook secret
			tempConfig := config
			tempConfig.WebhookSecret = ""
			tempProvider := msgxwhatsapp.NewWhatsAppProvider(tempConfig)

			incomingMsg, err := tempProvider.HandleWebhook(r.Context(), r)
			if err != nil {
				log.Printf("Error processing webhook: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if incomingMsg != nil {
				log.Printf("Message received (no verification): %+v", incomingMsg)
				if messageProcessor != nil {
					messageProcessor(incomingMsg)
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	log.Println("Server starting on :8080")
	log.Println("Endpoints available:")
	log.Println("  - GET|POST /webhook/whatsapp          - Production webhook")
	log.Println("  - GET|POST /webhook/whatsapp/debug    - Debug webhook")
	log.Println("  - GET|POST /webhook/whatsapp/verify   - Test verification")
	log.Println("  - GET|POST /webhook/whatsapp/no-verify - Skip signature verification")
	log.Println("  - GET      /test-signature            - Test signature generation")
	log.Println("  - GET      /health                    - Health check")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

