package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/ai/llm/guardrailx"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/craftable/ai/llm/toolx"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	// Create the LLM client
	provider := aiopenai.NewOpenAIProvider(apiKey)
	llmClient := llm.NewClient(provider)

	// Create tools
	weatherTool := NewWeatherTool()
	tools := toolx.FromToolx(weatherTool)

	// Create memory
	mem := memoryx.NewMemory(memoryx.WithSystemPrompt("You are a helpful assistant that can check weather conditions."))

	// Create the agent
	myAgent := agentx.New(
		*llmClient,
		mem,
		agentx.WithTools(tools),
		agentx.WithOptions(
			llm.WithModel("gpt-4o"),
			llm.WithMaxTokens(500),
		),
	)

	// Define guardrail rules
	rules := []guardrailx.Rule{
		{
			Name:        "Profanity Filter",
			Type:        guardrailx.BlocklistRule,
			Pattern:     []string{"damn", "hell", "shit", "fuck"},
			Direction:   guardrailx.Both,
			Action:      guardrailx.FilterAction,
			Replacement: "****",
			Message:     "Please avoid using profanity.",
			Priority:    90,
		},
		// LLM Rule to check if input is weather-related
		{
			Name:      "Weather Topic Classifier",
			Type:      guardrailx.LLMRule,
			Direction: guardrailx.Input,
			Action:    guardrailx.BlockAction,
			Message:   "I'm a weather assistant. Please ask me questions about weather, climate, or meteorological conditions.",
			Priority:  100, // Higher priority than the keyword check
			LLMConfig: &guardrailx.LLMCheckConfig{
				Client: llmClient,
				InputPrompt: `Determine if the following user query is related to weather, climate, or meteorological information:

{{content}}

A query is weather-related if it asks about:
- Current weather conditions in any location
- Weather forecasts or predictions
- Temperature, precipitation, humidity, wind, air pressure
- Climate patterns or historical weather data
- Meteorological phenomena (storms, hurricanes, etc.)
- Any conditions related to the atmosphere

Examples of weather-related queries:
- "What's the weather in Paris?"
- "Will it rain tomorrow in Seattle?"
- "Is it hot in Arizona right now?"
- "What's the forecast for this weekend?"
- "How's the climate in Peru?"
- "Tell me about hurricane formation"

Respond with ONLY "yes" if the query is related to weather or "no" if it's about something else.`,
				ModelOptions: []llm.Option{
					llm.WithModel("gpt-4o-mini"), // Use smaller model for guardrails to save costs
					llm.WithTemperature(0.0),     // Low temperature for consistent moderation
					llm.WithMaxTokens(10),        // Only need a short response
				},
				ValidResponse: []string{"yes"}, // Only "yes" is considered valid
			},
		},
	}

	// Create guardrail
	guardrail := guardrailx.NewGuardRail(
		rules,
		guardrailx.WithLogger(&CustomLogger{}),
		guardrailx.WithDefaultLLM(llmClient, []llm.Option{
			llm.WithModel("gpt-3.5-turbo"),
			llm.WithTemperature(0.0),
		}),
	)

	// Create guarded agent
	guardedAgent := guardrailx.NewGuardedAgent(myAgent, guardrail, nil)

	// Interactive chat loop
	fmt.Println("=== Weather Assistant with LLM Guardrails ===")
	fmt.Println("Type your questions (press Ctrl+C to exit)")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if strings.TrimSpace(input) == "" {
			continue
		}

		response, err := guardedAgent.Run(context.Background(), input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println("\nA:", response)
	}
}

// CustomLogger logs guardrail violations
type CustomLogger struct{}

func (l *CustomLogger) Log(rule guardrailx.Rule, content string, direction guardrailx.Direction, result guardrailx.ValidationResult) error {
	fmt.Printf("[GUARDRAIL] %s rule '%s' triggered for %s message\n", rule.Type, rule.Name, direction)
	if result.Details != "" {
		fmt.Printf("  LLM response: %s\n", result.Details)
	}
	return nil
}

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
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
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

	// Fixed temperature symbol
	weatherData := fmt.Sprintf("Currently 22°C and partly cloudy in %s.", request.Location)
	return weatherData, nil
}
