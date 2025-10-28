package llm

// ChatOptions contains options for generating chat completions
type ChatOptions struct {
	Model            string            // Model name/identifier
	Temperature      float32           // Controls randomness (0.0 to 1.0) - not supported by o-series models
	TopP             float32           // Controls diversity (0.0 to 1.0) - not supported by o-series models
	MaxTokens        int               // Maximum number of tokens to generate (max_completion_tokens for o-series)
	Stop             []string          // Stop sequences
	Tools            []Tool            // Available tools
	Functions        []Function        // Available functions for backward compatibility
	ToolChoice       any               // Force specific tool
	ResponseFormat   *ResponseFormat   // Response format specification
	PresencePenalty  float32           // Penalty for new tokens based on presence - not supported by o-series models
	FrequencyPenalty float32           // Penalty for new tokens based on frequency - not supported by o-series models
	LogitBias        map[int]float32   // Modify the likelihood of specified tokens appearing
	Seed             int64             // Random seed for deterministic results
	Stream           bool              // Whether to stream the response
	User             string            // Identifier representing end-user
	JSONMode         bool              // Shorthand for JSON response format
	Headers          map[string]string // Custom headers to send with the request
	ReasoningEffort  string            // For o-series models: "low", "medium", or "high" - controls reasoning depth
	ParallelToolCalls *bool            // Whether to allow parallel tool calls (default: true)
	StoreOutput      *bool             // Whether to store the output for training (default: follows org settings)
	Metadata         map[string]string // Metadata to attach to the request
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

// WithMaxTokens sets the maximum number of tokens to generate
func WithMaxTokens(tokens int) Option {
	return func(o *ChatOptions) {
		o.MaxTokens = tokens
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

// WithReasoningEffort sets the reasoning effort level for o-series models
// Valid values: "low", "medium", "high"
func WithReasoningEffort(effort string) Option {
	return func(o *ChatOptions) {
		o.ReasoningEffort = effort
	}
}

// WithParallelToolCalls enables or disables parallel tool calls
func WithParallelToolCalls(enabled bool) Option {
	return func(o *ChatOptions) {
		o.ParallelToolCalls = &enabled
	}
}

// WithStoreOutput sets whether to store the output for training
func WithStoreOutput(store bool) Option {
	return func(o *ChatOptions) {
		o.StoreOutput = &store
	}
}

// WithMetadata adds metadata to the request
func WithMetadata(metadata map[string]string) Option {
	return func(o *ChatOptions) {
		o.Metadata = metadata
	}
}

// WithLogitBias sets the logit bias map
func WithLogitBias(bias map[int]float32) Option {
	return func(o *ChatOptions) {
		o.LogitBias = bias
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
