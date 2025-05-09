package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Abraxas-365/craftale/ai/llm"
	"github.com/openai/openai-go"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	// Create the OpenAI provider
	provider := llm.NewOpenAIProvider(apiKey)

	// Create a client with the provider
	client := llm.NewClient(provider)

	// Run examples
	fmt.Println("=== Basic Chat Example ===")
	basicChatExample(client)

	fmt.Println("\n=== Streaming Chat Example ===")
	streamingChatExample(client)

	fmt.Println("\n=== Tool Call Example ===")
	toolCallExample(client)
}

func basicChatExample(client *llm.Client) {
	// Create a conversation
	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant that provides concise answers."),
		llm.NewUserMessage("What's the capital of France?"),
	}

	// Get a response
	resp, err := client.Chat(context.Background(), messages,
		llm.WithModel(openai.ChatModelGPT4o),
		llm.WithTemperature(0.7),
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the response
	fmt.Println("Assistant:", resp.Message.Content)
	fmt.Printf("Usage: %d tokens total (%d prompt, %d completion)\n",
		resp.Usage.TotalTokens,
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens)

	// Continue the conversation
	messages = append(messages, resp.Message) // Add assistant's response
	messages = append(messages, llm.NewUserMessage("What's its population?"))

	// Get second response
	resp2, err := client.Chat(context.Background(), messages,
		llm.WithModel(openai.ChatModelGPT4o),
		llm.WithTemperature(0.7),
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print second response
	fmt.Println("Assistant:", resp2.Message.Content)
}

func streamingChatExample(client *llm.Client) {
	// Create a conversation
	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant that generates creative content."),
		llm.NewUserMessage("Write a short haiku about programming in Go"),
	}

	// Get a streaming response
	stream, err := client.ChatStream(context.Background(), messages,
		llm.WithModel(openai.ChatModelGPT4o),
		llm.WithTemperature(0.8),
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Print("Assistant: ")

	// Collect the full response for later use
	var fullResponse strings.Builder

	// Print the streaming response
	for {
		msg, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			break
		}

		fmt.Print(msg.Content)
		fullResponse.WriteString(msg.Content)
	}

	fmt.Println()
	stream.Close()
}

func toolCallExample(client *llm.Client) {
	// Define weather tool
	weatherTool := llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        "get_weather",
			Description: "Get the current weather in a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and state, e.g. San Francisco, CA",
					},
					"unit": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"celsius", "fahrenheit"},
						"description": "The temperature unit to use",
					},
				},
				"required": []string{"location"},
			},
		},
	}
	// Create a conversation with a query that should trigger the tool
	messages := []llm.Message{
		llm.NewUserMessage("What's the weather like in Paris today?"),
	}

	// Make the request with the tool
	resp, err := client.Chat(context.Background(), messages,
		llm.WithModel(openai.ChatModelGPT4o),
		llm.WithTools([]llm.Tool{weatherTool}),
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check if we got a tool call
	if len(resp.Message.ToolCalls) > 0 {
		fmt.Println("Assistant wants to use a tool:")

		for _, toolCall := range resp.Message.ToolCalls {
			fmt.Printf("Tool: %s\n", toolCall.Function.Name)
			fmt.Printf("Arguments: %s\n", toolCall.Function.Arguments)

			// Simulate getting weather data
			weatherData := "It's 24°C and sunny in Paris"

			// Add the assistant's response and our tool response to the conversation
			messages = append(messages, resp.Message)
			messages = append(messages, llm.NewToolMessage(toolCall.ID, weatherData))

			// Get the final response
			finalResp, err := client.Chat(context.Background(), messages,
				llm.WithModel(openai.ChatModelGPT4o),
			)

			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			// Print the final response
			fmt.Println("\nAssistant's final response:")
			fmt.Println(finalResp.Message.Content)
		}
	} else {
		// Print the response if no tool calls
		fmt.Println("Assistant:", resp.Message.Content)
	}
}
