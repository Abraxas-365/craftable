package llm

import "strings"

// OpenAI Model Constants
const (
	// GPT-4o models (multimodal, latest generation)
	ModelGPT4o       = "gpt-4o"
	ModelGPT4oMini   = "gpt-4o-mini"
	ModelGPT4o_2024  = "gpt-4o-2024-11-20"
	ModelGPT4o_Audio = "gpt-4o-audio-preview"

	// GPT-4 Turbo models
	ModelGPT4Turbo         = "gpt-4-turbo"
	ModelGPT4TurboPreview  = "gpt-4-turbo-preview"
	ModelGPT4_0125_Preview = "gpt-4-0125-preview"
	ModelGPT4_1106_Preview = "gpt-4-1106-preview"

	// GPT-4 models
	ModelGPT4               = "gpt-4"
	ModelGPT4_0613          = "gpt-4-0613"
	ModelGPT4_32k           = "gpt-4-32k"
	ModelGPT4_32k_0613      = "gpt-4-32k-0613"
	ModelGPT4VisionPreview  = "gpt-4-vision-preview"

	// GPT-3.5 Turbo models
	ModelGPT3_5Turbo      = "gpt-3.5-turbo"
	ModelGPT3_5Turbo_0125 = "gpt-3.5-turbo-0125"
	ModelGPT3_5Turbo_1106 = "gpt-3.5-turbo-1106"
	ModelGPT3_5Turbo_16k  = "gpt-3.5-turbo-16k"

	// o-series models (reasoning models)
	ModelO1        = "o1"
	ModelO1_2024   = "o1-2024-12-17"
	ModelO1Preview = "o1-preview"
	ModelO1Mini    = "o1-mini"
	ModelO3Mini    = "o3-mini"
)

// Reasoning effort levels for o-series models
const (
	ReasoningEffortLow    = "low"
	ReasoningEffortMedium = "medium"
	ReasoningEffortHigh   = "high"
)

// ModelCapabilities defines what features a model supports
type ModelCapabilities struct {
	SupportsTemperature      bool
	SupportsTopP             bool
	SupportsPenalties        bool // presence_penalty, frequency_penalty
	SupportsStreaming        bool
	SupportsTools            bool
	SupportsVision           bool
	SupportsReasoningEffort  bool
	IsReasoningModel         bool
	RequiresDeveloperMessage bool // o-series uses developer messages instead of system
}

// GetModelCapabilities returns the capabilities for a given model
func GetModelCapabilities(model string) ModelCapabilities {
	modelLower := strings.ToLower(model)

	// o-series models have special capabilities
	if IsReasoningModel(model) {
		return ModelCapabilities{
			SupportsTemperature:      false,
			SupportsTopP:             false,
			SupportsPenalties:        false,
			SupportsStreaming:        true, // o3-mini and newer support streaming
			SupportsTools:            true,
			SupportsVision:           false,
			SupportsReasoningEffort:  true,
			IsReasoningModel:         true,
			RequiresDeveloperMessage: true,
		}
	}

	// GPT-4o and GPT-4 models
	if strings.Contains(modelLower, "gpt-4o") || strings.Contains(modelLower, "gpt-4") {
		return ModelCapabilities{
			SupportsTemperature: true,
			SupportsTopP:        true,
			SupportsPenalties:   true,
			SupportsStreaming:   true,
			SupportsTools:       true,
			SupportsVision:      strings.Contains(modelLower, "vision") || strings.Contains(modelLower, "gpt-4o"),
			IsReasoningModel:    false,
		}
	}

	// GPT-3.5 models
	if strings.Contains(modelLower, "gpt-3.5") {
		return ModelCapabilities{
			SupportsTemperature: true,
			SupportsTopP:        true,
			SupportsPenalties:   true,
			SupportsStreaming:   true,
			SupportsTools:       true,
			SupportsVision:      false,
			IsReasoningModel:    false,
		}
	}

	// Default capabilities (assume modern model)
	return ModelCapabilities{
		SupportsTemperature: true,
		SupportsTopP:        true,
		SupportsPenalties:   true,
		SupportsStreaming:   true,
		SupportsTools:       true,
		SupportsVision:      false,
		IsReasoningModel:    false,
	}
}

// IsReasoningModel checks if the model is an o-series reasoning model
func IsReasoningModel(model string) bool {
	modelLower := strings.ToLower(model)
	return strings.HasPrefix(modelLower, "o1") ||
		strings.HasPrefix(modelLower, "o3") ||
		strings.Contains(modelLower, "o1-") ||
		strings.Contains(modelLower, "o3-")
}

// IsVisionModel checks if the model supports vision/image inputs
func IsVisionModel(model string) bool {
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "vision") ||
		strings.Contains(modelLower, "gpt-4o")
}

// ValidateOptionsForModel validates options against model capabilities
// Returns a list of warnings for unsupported options
func ValidateOptionsForModel(model string, opts *ChatOptions) []string {
	capabilities := GetModelCapabilities(model)
	warnings := make([]string, 0)

	if !capabilities.SupportsTemperature && opts.Temperature != 0 {
		warnings = append(warnings, "temperature parameter is not supported by "+model)
	}

	if !capabilities.SupportsTopP && opts.TopP != 0 && opts.TopP != 1.0 {
		warnings = append(warnings, "top_p parameter is not supported by "+model)
	}

	if !capabilities.SupportsPenalties {
		if opts.PresencePenalty != 0 {
			warnings = append(warnings, "presence_penalty parameter is not supported by "+model)
		}
		if opts.FrequencyPenalty != 0 {
			warnings = append(warnings, "frequency_penalty parameter is not supported by "+model)
		}
	}

	if !capabilities.SupportsStreaming && opts.Stream {
		warnings = append(warnings, "streaming is not supported by "+model)
	}

	if !capabilities.SupportsReasoningEffort && opts.ReasoningEffort != "" {
		warnings = append(warnings, "reasoning_effort parameter is not supported by "+model)
	}

	return warnings
}

// SanitizeOptionsForModel removes or adjusts options that are not supported by the model
// This is useful to prevent API errors
func SanitizeOptionsForModel(model string, opts *ChatOptions) {
	capabilities := GetModelCapabilities(model)

	if !capabilities.SupportsTemperature {
		opts.Temperature = 0
	}

	if !capabilities.SupportsTopP {
		opts.TopP = 0
	}

	if !capabilities.SupportsPenalties {
		opts.PresencePenalty = 0
		opts.FrequencyPenalty = 0
	}

	if !capabilities.SupportsReasoningEffort {
		opts.ReasoningEffort = ""
	}
}

// GetDefaultModel returns the default model for general use
func GetDefaultModel() string {
	return ModelGPT4o
}

// GetDefaultReasoningModel returns the default o-series model
func GetDefaultReasoningModel() string {
	return ModelO3Mini
}
