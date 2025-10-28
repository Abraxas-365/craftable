package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI provider
	provider := aiopenai.NewOpenAIProvider(apiKey)
	client := llm.NewClient(provider)

	ctx := context.Background()

	fmt.Println("=== AI Package with o-series Model Support Examples ===\n")

	// Example 1: Using o3-mini with reasoning effort
	fmt.Println("1. o3-mini with reasoning effort:")
	example1(ctx, client)

	// Example 2: Using o3-mini with streaming
	fmt.Println("\n2. o3-mini with streaming:")
	example2(ctx, client)

	// Example 3: Using o-series with tools
	fmt.Println("\n3. o3-mini with tools/functions:")
	example3(ctx, client)

	// Example 4: Comparing GPT-4o with o3-mini
	fmt.Println("\n4. Comparing GPT-4o with o3-mini:")
	example4(ctx, client)

	// Example 5: Model capabilities and validation
	fmt.Println("\n5. Model capabilities and validation:")
	example5()

	// Example 6: Using new parameters (parallel tool calls, metadata, etc.)
	fmt.Println("\n6. Using new parameters:")
	example6(ctx, client)
}

// Example 1: Using o3-mini with reasoning effort
func example1(ctx context.Context, client *llm.Client) {
	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant that solves complex problems."),
		llm.NewUserMessage("What is the sum of all prime numbers between 1 and 100?"),
	}

	response, err := client.Chat(ctx, messages,
		llm.WithModel(llm.ModelO3Mini),
		llm.WithReasoningEffort(llm.ReasoningEffortMedium),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response.Message.Content)
	fmt.Printf("Usage - Prompt: %d, Completion: %d, Reasoning: %d, Total: %d\n",
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
		response.Usage.ReasoningTokens,
		response.Usage.TotalTokens,
	)
}

// Example 2: Using o3-mini with streaming
func example2(ctx context.Context, client *llm.Client) {
	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant."),
		llm.NewUserMessage("Explain quantum entanglement in simple terms."),
	}

	stream, err := client.ChatStream(ctx, messages,
		llm.WithModel(llm.ModelO3Mini),
		llm.WithReasoningEffort(llm.ReasoningEffortLow),
		llm.WithMaxTokens(500),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	defer stream.Close()

	fmt.Print("Response (streaming): ")
	var lastMessage llm.Message
	for {
		message, err := stream.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				lastMessage = message
				break
			}
			log.Printf("Error reading stream: %v\n", err)
			return
		}
		fmt.Print(message.Content[len(lastMessage.Content):])
		lastMessage = message
	}

	// Print usage information if available
	if usage, ok := lastMessage.Metadata["usage"].(llm.Usage); ok {
		fmt.Printf("\nUsage - Prompt: %d, Completion: %d, Reasoning: %d, Total: %d\n",
			usage.PromptTokens,
			usage.CompletionTokens,
			usage.ReasoningTokens,
			usage.TotalTokens,
		)
	}
	fmt.Println()
}

// Example 3: Using o-series with tools
func example3(ctx context.Context, client *llm.Client) {
	// Define a simple calculator tool
	calculatorTool := llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        "calculate",
			Description: "Perform basic arithmetic operations",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "The operation to perform: add, subtract, multiply, divide",
						"enum":        []string{"add", "subtract", "multiply", "divide"},
					},
					"a": map[string]any{
						"type":        "number",
						"description": "The first number",
					},
					"b": map[string]any{
						"type":        "number",
						"description": "The second number",
					},
				},
				"required": []string{"operation", "a", "b"},
			},
		},
	}

	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant with access to a calculator."),
		llm.NewUserMessage("What is 156 * 234?"),
	}

	response, err := client.Chat(ctx, messages,
		llm.WithModel(llm.ModelO3Mini),
		llm.WithTools([]llm.Tool{calculatorTool}),
		llm.WithReasoningEffort(llm.ReasoningEffortLow),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	if len(response.Message.ToolCalls) > 0 {
		fmt.Printf("Tool called: %s\n", response.Message.ToolCalls[0].Function.Name)
		fmt.Printf("Arguments: %s\n", response.Message.ToolCalls[0].Function.Arguments)
	} else {
		fmt.Printf("Response: %s\n", response.Message.Content)
	}

	fmt.Printf("Reasoning tokens used: %d\n", response.Usage.ReasoningTokens)
}

// Example 4: Comparing GPT-4o with o3-mini
func example4(ctx context.Context, client *llm.Client) {
	prompt := "Solve this logic puzzle: Three friends are named Alex, Ben, and Charlie. Alex is taller than Ben, and Charlie is shorter than Ben. Who is the shortest?"

	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant."),
		llm.NewUserMessage(prompt),
	}

	// GPT-4o
	fmt.Println("GPT-4o response:")
	response1, err := client.Chat(ctx, messages,
		llm.WithModel(llm.ModelGPT4o),
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  %s\n", response1.Message.Content)
		fmt.Printf("  Tokens: %d\n", response1.Usage.TotalTokens)
	}

	// o3-mini
	fmt.Println("\no3-mini response:")
	response2, err := client.Chat(ctx, messages,
		llm.WithModel(llm.ModelO3Mini),
		llm.WithReasoningEffort(llm.ReasoningEffortMedium),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  %s\n", response2.Message.Content)
		fmt.Printf("  Total tokens: %d (Reasoning: %d)\n",
			response2.Usage.TotalTokens,
			response2.Usage.ReasoningTokens,
		)
	}
}

// Example 5: Model capabilities and validation
func example5() {
	models := []string{
		llm.ModelGPT4o,
		llm.ModelO3Mini,
		llm.ModelO1Preview,
		llm.ModelGPT3_5Turbo,
	}

	fmt.Println("Model Capabilities:")
	for _, model := range models {
		caps := llm.GetModelCapabilities(model)
		fmt.Printf("\n%s:\n", model)
		fmt.Printf("  - Supports Temperature: %v\n", caps.SupportsTemperature)
		fmt.Printf("  - Supports Top-P: %v\n", caps.SupportsTopP)
		fmt.Printf("  - Supports Penalties: %v\n", caps.SupportsPenalties)
		fmt.Printf("  - Supports Streaming: %v\n", caps.SupportsStreaming)
		fmt.Printf("  - Supports Tools: %v\n", caps.SupportsTools)
		fmt.Printf("  - Supports Reasoning Effort: %v\n", caps.SupportsReasoningEffort)
		fmt.Printf("  - Is Reasoning Model: %v\n", caps.IsReasoningModel)
	}

	// Validate options for o3-mini
	fmt.Println("\nValidation example - o3-mini with invalid options:")
	options := &llm.ChatOptions{
		Model:            llm.ModelO3Mini,
		Temperature:      0.7,
		TopP:             0.9,
		PresencePenalty:  0.5,
		FrequencyPenalty: 0.3,
		ReasoningEffort:  llm.ReasoningEffortHigh,
	}

	warnings := llm.ValidateOptionsForModel(llm.ModelO3Mini, options)
	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	// Sanitize options
	llm.SanitizeOptionsForModel(llm.ModelO3Mini, options)
	fmt.Printf("After sanitization:\n")
	fmt.Printf("  - Temperature: %.2f (should be 0)\n", options.Temperature)
	fmt.Printf("  - TopP: %.2f (should be 0)\n", options.TopP)
	fmt.Printf("  - ReasoningEffort: %s (preserved)\n", options.ReasoningEffort)
}

// Example 6: Using new parameters
func example6(ctx context.Context, client *llm.Client) {
	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant."),
		llm.NewUserMessage("What's 2+2?"),
	}

	// Use new parameters
	response, err := client.Chat(ctx, messages,
		llm.WithModel(llm.ModelGPT4o),
		llm.WithParallelToolCalls(true),
		llm.WithMetadata(map[string]string{
			"user_id":    "12345",
			"session_id": "abc-def-ghi",
		}),
		llm.WithSeed(42), // For reproducibility
		llm.WithLogitBias(map[int]float32{
			// Token IDs and biases (example)
		}),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response.Message.Content)
	fmt.Printf("Usage: Prompt=%d, Completion=%d, Total=%d\n",
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
		response.Usage.TotalTokens,
	)

	// Show completion tokens details if available
	if response.Usage.CompletionTokensDetails != nil {
		details := response.Usage.CompletionTokensDetails
		fmt.Printf("Completion Details:\n")
		fmt.Printf("  - Reasoning tokens: %d\n", details.ReasoningTokens)
		fmt.Printf("  - Accepted prediction tokens: %d\n", details.AcceptedPredictionTokens)
		fmt.Printf("  - Rejected prediction tokens: %d\n", details.RejectedPredictionTokens)
	}
}
