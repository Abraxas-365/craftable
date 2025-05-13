package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Abraxas-365/craftable/ai/aiproviders"
	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/craftable/ai/llm/toolx"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	// Create the LLM client using your existing provider
	provider := aiproviders.NewOpenAIProvider(apiKey)
	client := llm.NewClient(provider)

	// Create a simple tool
	weatherTool := NewWeatherTool()
	tools := toolx.FromToolx(weatherTool)

	// Create memory with system prompt
	mem := memoryx.NewMemory("You are a helpful assistant that can check weather conditions.")

	// Create the agent
	myAgent := agentx.New(
		client,
		mem,
		agentx.WithTools(tools),
		agentx.WithOptions(
			llm.WithModel("gpt-4o"),
			llm.WithMaxTokens(500),
			llm.WithTemperature(0.7),
		),
	)

	// Use the agent.EvaluateWithTools function for a detailed execution trace
	userQuery := "What's the weather like in New York?"
	fmt.Println("Running agent with query:", userQuery)

	eval, err := myAgent.EvaluateWithTools(context.Background(), userQuery)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the detailed execution trace
	fmt.Println("\n=== Agent Evaluation Results ===")
	fmt.Println("User Query:", eval.UserInput)

	fmt.Println("\n--- Execution Steps ---")
	for i, step := range eval.Steps {
		fmt.Printf("\nStep %d (%s):\n", i+1, step.StepType)

		if step.StepType == "initial" || step.StepType == "response" {
			fmt.Printf("LLM Output: %s\n", step.OutputMessage.Content)

			if len(step.OutputMessage.ToolCalls) > 0 {
				fmt.Println("Tool Calls:")
				for _, tc := range step.OutputMessage.ToolCalls {
					fmt.Printf("  - %s: %s\n", tc.Function.Name, tc.Function.Arguments)
				}
			}

			fmt.Printf("Tokens Used: %d\n", step.TokenUsage.TotalTokens)
		}

		if step.StepType == "tool_execution" {
			fmt.Println("Tool Responses:")
			for _, tr := range step.ToolResponses {
				fmt.Printf("  - %s\n", tr.Content)
			}
		}
	}

	fmt.Println("\n--- Final Result ---")
	fmt.Println(eval.FinalResponse)
}

// WeatherTool provides weather information
type WeatherTool struct{}

type WeatherRequest struct {
	Location string `json:"location"`
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

func (w *WeatherTool) Name() string {
	return "get_weather"
}

func (w *WeatherTool) GetTool() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        w.Name(),
			Description: "Get the current weather in a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name, e.g. New York",
					},
				},
				"required": []string{"location"},
			},
		},
	}
}

func (w *WeatherTool) Call(ctx context.Context, inputs string) (any, error) {
	// Parse input
	var request WeatherRequest
	if err := json.Unmarshal([]byte(inputs), &request); err != nil {
		return nil, fmt.Errorf("failed to parse weather request: %w", err)
	}

	// Simple mock implementation - in a real app, you'd call a weather API
	weatherData := fmt.Sprintf("Currently 22°C and partly cloudy in %s.", request.Location)
	return weatherData, nil
}
