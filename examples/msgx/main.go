package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/configx"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/msgx"
	"github.com/Abraxas-365/craftable/msgx/msgxproviders"
)

func main() {
	// Set environment variables for demo
	os.Setenv("TWILIO_ACCOUNT_SID", "SID")
	os.Setenv("TWILIO_AUTH_TOKEN", "TOKEN")
	os.Setenv("TWILIO_FROM_NUMBER", "+16198485487")
	os.Setenv("TWILIO_TEST_PHONE_NUMBER", "+<numer>")

	// Create configuration
	config, err := configx.NewBuilder().
		WithDefaults(map[string]any{
			"twilio": map[string]any{
				"api": map[string]any{
					"version": "2010-04-01",
				},
				"http": map[string]any{
					"timeout": 30,
				},
			},
		}).
		FromEnv("TWILIO_").
		RequireEnv(
			"TWILIO_ACCOUNT_SID",
			"TWILIO_AUTH_TOKEN",
			"TWILIO_FROM_NUMBER",
			"TWILIO_TEST_PHONE_NUMBER",
		).
		Build()

	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	// Create Twilio provider using config
	twilioProvider := msgxproviders.NewTwilioProvider(msgxproviders.TwilioConfig{
		AccountSID:  config.Get("account.sid").AsString(),
		AuthToken:   config.Get("auth.token").AsString(),
		FromNumber:  config.Get("from.number").AsString(),
		APIVersion:  config.Get("api.version").AsString(),
		HTTPTimeout: config.Get("http.timeout").AsInt(),
	})

	// Send a simple message
	ctx := context.Background()
	message := msgx.Message{
		To:   config.Get("test.phone.number").AsString(),
		Type: msgx.MessageTypeText,
		Content: msgx.Content{
			Text: &msgx.TextContent{
				Body: "Hello from Twilio SMS with configx! 🚀",
			},
		},
	}

	response, err := twilioProvider.Send(ctx, message)
	if err != nil {
		log.Fatalf("Failed to send message: %v", errx.Print(err))
	}

	fmt.Printf("✅ Message sent successfully!\n")
	fmt.Printf("Message ID: %s\n", response.MessageID)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("To: %s\n", response.To)
}
