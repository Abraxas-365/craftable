package llm

// ChatOptions contains options for generating chat completions
type ChatOptions struct {
	Model               string            // Model name/identifier
	Temperature         float32           // Controls randomness (0.0 to 1.0)
	TopP                float32           // Controls diversity (0.0 to 1.0)
	MaxTokens           int               // Maximum number of tokens to generate (legacy)
	MaxCompletionTokens int               // Maximum completion tokens (preferred for new models)
	Stop                []string          // Stop sequences
	Tools               []Tool            // Available tools
	Functions           []Function        // Available functions for backward compatibility
	ToolChoice          any               // Force specific tool
	ResponseFormat      *ResponseFormat   // Response format specification
	PresencePenalty     float32           // Penalty for new tokens based on presence
	FrequencyPenalty    float32           // Penalty for new tokens based on frequency
	LogitBias           map[int]float32   // Modify the likelihood of specified tokens appearing
	Seed                int64             // Random seed for deterministic results
	Stream              bool              // Whether to stream the response
	User                string            // Identifier representing end-user
	JSONMode            bool              // Shorthand for JSON response format
	Headers             map[string]string // Custom headers to send with the request

	ReasoningEffort string // Reasoning effort level: "low", "medium", "high"

}

// Option is a function type to modify ChatOptions
type Option func(*ChatOptions)

// WithModel sets the model to use
func WithModel(model string) Option {
	return func(o *ChatOptions) {
		o.Model = model
	}
}

// WithTemperature sets the sampling temperature
func WithTemperature(temp float32) Option {
	return func(o *ChatOptions) {
		o.Temperature = temp
	}
}

// WithTopP sets nucleus sampling parameter
func WithTopP(topP float32) Option {
	return func(o *ChatOptions) {
		o.TopP = topP
	}
}

// WithMaxTokens sets the maximum number of tokens to generate (legacy)
func WithMaxTokens(tokens int) Option {
	return func(o *ChatOptions) {
		o.MaxTokens = tokens
	}
}

// WithMaxCompletionTokens sets the maximum completion tokens (preferred)
func WithMaxCompletionTokens(tokens int) Option {
	return func(o *ChatOptions) {
		o.MaxCompletionTokens = tokens
	}
}

// WithStop sets sequences where the API will stop generating further tokens
func WithStop(stop []string) Option {
	return func(o *ChatOptions) {
		o.Stop = stop
	}
}

// WithTools sets the available tools
func WithTools(tools []Tool) Option {
	return func(o *ChatOptions) {
		o.Tools = tools
	}
}

// WithFunctions sets the available functions (legacy approach)
func WithFunctions(functions []Function) Option {
	return func(o *ChatOptions) {
		o.Functions = functions
	}
}

// WithToolChoice forces a specific tool
func WithToolChoice(toolChoice any) Option {
	return func(o *ChatOptions) {
		o.ToolChoice = toolChoice
	}
}

// WithJSONMode enables JSON mode
func WithJSONMode() Option {
	return func(o *ChatOptions) {
		o.JSONMode = true
	}
}

// WithStream enables streaming response
func WithStream(stream bool) Option {
	return func(o *ChatOptions) {
		o.Stream = stream
	}
}

// WithHeader adds a custom header to the request
func WithHeader(key, value string) Option {
	return func(o *ChatOptions) {
		if o.Headers == nil {
			o.Headers = make(map[string]string)
		}
		o.Headers[key] = value
	}
}

// WithPresencePenalty sets the presence penalty
func WithPresencePenalty(penalty float32) Option {
	return func(o *ChatOptions) {
		o.PresencePenalty = penalty
	}
}

// WithFrequencyPenalty sets the frequency penalty
func WithFrequencyPenalty(penalty float32) Option {
	return func(o *ChatOptions) {
		o.FrequencyPenalty = penalty
	}
}

// WithSeed sets the random seed
func WithSeed(seed int64) Option {
	return func(o *ChatOptions) {
		o.Seed = seed
	}
}

// WithUser sets the user identifier
func WithUser(user string) Option {
	return func(o *ChatOptions) {
		o.User = user
	}
}

// WithReasoningEffort sets the reasoning effort for reasoning models (o1, o3)
// Valid values: "low", "medium", "high"
func WithReasoningEffort(effort string) Option {
	return func(o *ChatOptions) {
		o.ReasoningEffort = effort
	}
}

// DefaultOptions returns the default options
func DefaultOptions() *ChatOptions {
	return &ChatOptions{
		Temperature: 0.7,
		TopP:        1.0,
		MaxTokens:   0, // No limit by default
	}
}
