# AI Package Examples

This directory contains examples demonstrating the capabilities of the Craftable AI package.

## Overview

The Craftable AI package provides a unified interface for working with various AI models, with first-class support for:
- OpenAI GPT models (GPT-4o, GPT-4, GPT-3.5)
- OpenAI o-series reasoning models (o1, o3-mini)
- Streaming responses with token usage tracking
- Tool/function calling
- Model-specific parameter validation

## New Features (o-series Support)

### 1. o-series Reasoning Models

The package now fully supports OpenAI's o-series models (o1, o1-preview, o1-mini, o3-mini) with their unique capabilities:

```go
response, err := client.Chat(ctx, messages,
    llm.WithModel(llm.ModelO3Mini),
    llm.WithReasoningEffort(llm.ReasoningEffortMedium),
    llm.WithMaxTokens(2000),
)
```

**Reasoning Effort Levels:**
- `ReasoningEffortLow` - Faster, less reasoning
- `ReasoningEffortMedium` - Balanced
- `ReasoningEffortHigh` - More thorough reasoning

### 2. Reasoning Token Tracking

o-series models provide detailed token usage including reasoning tokens:

```go
fmt.Printf("Usage:\n")
fmt.Printf("  Prompt tokens: %d\n", response.Usage.PromptTokens)
fmt.Printf("  Completion tokens: %d\n", response.Usage.CompletionTokens)
fmt.Printf("  Reasoning tokens: %d\n", response.Usage.ReasoningTokens)
fmt.Printf("  Total tokens: %d\n", response.Usage.TotalTokens)
```

### 3. Model Capabilities System

Automatic detection and validation of model capabilities:

```go
// Check what a model supports
caps := llm.GetModelCapabilities(llm.ModelO3Mini)
fmt.Printf("Supports temperature: %v\n", caps.SupportsTemperature)
fmt.Printf("Supports reasoning effort: %v\n", caps.SupportsReasoningEffort)
fmt.Printf("Is reasoning model: %v\n", caps.IsReasoningModel)

// Validate options before using
warnings := llm.ValidateOptionsForModel(model, options)

// Automatically sanitize options
llm.SanitizeOptionsForModel(model, options)
```

### 4. Enhanced Streaming Support

Streaming now includes usage information in metadata:

```go
stream, err := client.ChatStream(ctx, messages,
    llm.WithModel(llm.ModelO3Mini),
    llm.WithReasoningEffort(llm.ReasoningEffortLow),
)

for {
    message, err := stream.Next()
    if errors.Is(err, io.EOF) {
        // Get usage from metadata
        if usage, ok := message.Metadata["usage"].(llm.Usage); ok {
            fmt.Printf("Reasoning tokens: %d\n", usage.ReasoningTokens)
        }
        break
    }
    fmt.Print(message.Content)
}
```

### 5. New Parameters

Additional parameters for fine-grained control:

```go
response, err := client.Chat(ctx, messages,
    // Control parallel tool execution
    llm.WithParallelToolCalls(true),

    // Add custom metadata to requests
    llm.WithMetadata(map[string]string{
        "user_id": "12345",
        "session_id": "abc-def",
    }),

    // Set seed for reproducibility
    llm.WithSeed(42),

    // Modify token likelihoods
    llm.WithLogitBias(map[int]float32{
        1234: 0.5,
    }),

    // Control output storage
    llm.WithStoreOutput(false),
)
```

### 6. Developer Messages for o-series

o-series models use "developer" messages instead of "system" messages. This is handled automatically:

```go
// This works for both GPT models and o-series models
messages := []llm.Message{
    llm.NewSystemMessage("You are a helpful assistant."), // Converted to developer for o-series
    llm.NewUserMessage("Hello!"),
}
```

## Model Constants

The package provides constants for all supported models:

```go
// GPT-4o models
llm.ModelGPT4o
llm.ModelGPT4oMini
llm.ModelGPT4o_2024
llm.ModelGPT4o_Audio

// GPT-4 models
llm.ModelGPT4
llm.ModelGPT4Turbo
llm.ModelGPT4VisionPreview

// GPT-3.5 models
llm.ModelGPT3_5Turbo

// o-series models
llm.ModelO1
llm.ModelO1_2024
llm.ModelO1Preview
llm.ModelO1Mini
llm.ModelO3Mini
```

## Key Differences: GPT-4o vs o-series

| Feature | GPT-4o | o-series |
|---------|--------|----------|
| Temperature | ✅ Supported | ❌ Not supported |
| Top-P | ✅ Supported | ❌ Not supported |
| Presence Penalty | ✅ Supported | ❌ Not supported |
| Frequency Penalty | ✅ Supported | ❌ Not supported |
| Reasoning Effort | ❌ Not supported | ✅ Supported |
| Reasoning Tokens | ❌ Not tracked | ✅ Tracked |
| System Messages | ✅ System role | ✅ Developer role |
| Streaming | ✅ Supported | ✅ Supported |
| Tools/Functions | ✅ Supported | ✅ Supported |
| Vision | ✅ Supported | ❌ Not supported |

## Running the Examples

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="your-api-key-here"

# Run the comprehensive example
go run examples/o_series_example.go
```

## Example Output

```
=== AI Package with o-series Model Support Examples ===

1. o3-mini with reasoning effort:
Response: The sum of all prime numbers between 1 and 100 is 1060.
Usage - Prompt: 35, Completion: 18, Reasoning: 256, Total: 309

2. o3-mini with streaming:
Response (streaming): Quantum entanglement is a phenomenon where...
Usage - Prompt: 28, Completion: 95, Reasoning: 128, Total: 251

...
```

## Best Practices

### 1. Choose the Right Model

- **GPT-4o**: Best for general tasks, vision, and when you need fine control (temperature, etc.)
- **o3-mini**: Best for reasoning tasks, math, logic puzzles, complex analysis
- **o1/o1-preview**: Best for very complex reasoning (more expensive)

### 2. Use Reasoning Effort Wisely

- **Low**: Simple questions, quick responses needed
- **Medium**: Balanced - good default for most reasoning tasks
- **High**: Complex problems requiring deep analysis (more expensive)

### 3. Handle Model Differences

Always use the capability detection system when supporting multiple models:

```go
caps := llm.GetModelCapabilities(model)

if caps.IsReasoningModel {
    // Use reasoning-specific parameters
    opts = append(opts, llm.WithReasoningEffort(llm.ReasoningEffortMedium))
} else {
    // Use standard parameters
    opts = append(opts, llm.WithTemperature(0.7))
}
```

### 4. Monitor Token Usage

o-series models can use significant reasoning tokens. Always check usage:

```go
if response.Usage.ReasoningTokens > 0 {
    fmt.Printf("Reasoning tokens: %d\n", response.Usage.ReasoningTokens)
    // Consider adjusting reasoning_effort if too high
}
```

## Architecture

The package follows a clean architecture:

```
ai/
├── llm/                    # Core LLM interface and types
│   ├── llm.go             # Main interface
│   ├── models.go          # Message and usage types
│   ├── options.go         # Configuration options
│   ├── model_constants.go # Model definitions and capabilities
│   └── agentx/            # Agent framework
├── providers/             # Provider implementations
│   └── aiopenai/         # OpenAI provider
├── embedding/            # Embedding functionality
├── ocr/                  # Vision/OCR
├── speech/               # TTS and STT
└── examples/             # This directory
```

## Further Reading

- [OpenAI o-series Documentation](https://platform.openai.com/docs/models/o-series)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Reasoning Models Best Practices](https://platform.openai.com/docs/guides/reasoning)

## Support

For issues or questions, please open an issue in the repository.
