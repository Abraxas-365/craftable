package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/ai/aiproviders"
	"github.com/Abraxas-365/craftable/ai/ocr"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create a new OpenAI provider
	provider := aiproviders.NewOpenAIProvider(apiKey)

	// Create a context
	ctx := context.Background()

	// Use a public image URL instead of a local file
	imageURL := "https://user-images.githubusercontent.com/8736329/70234375-886b2080-1786-11ea-9f66-b68dfb759dcb.png"

	// Extract text from the image URL
	result, err := provider.ExtractTextFromURL(
		ctx,
		imageURL,
		ocr.WithModel("gpt-4.1-mini"),
		ocr.WithLanguage("auto"),
		ocr.WithDetailsLevel("high"),
	)
	if err != nil {
		log.Fatalf("Error extracting text from URL: %v", err)
	}

	fmt.Printf("Extracted Text:\n%s\n\n", result.Text)
	fmt.Printf("Overall Confidence: %.2f\n", result.Confidence)
	fmt.Printf("Number of Text Blocks: %d\n", len(result.Blocks))
	fmt.Printf("Processing Time: %d ms\n", result.Usage.ProcessingTime)
	fmt.Printf("Token Usage: %+v\n", result.Usage)
}
