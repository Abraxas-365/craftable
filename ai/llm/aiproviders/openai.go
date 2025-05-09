package aiproviders

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/Abraxas-365/craftale/ai/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

// OpenAIProvider implements the LLM interface for OpenAI
type OpenAIProvider struct {
	client openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, opts ...option.RequestOption) *OpenAIProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	options := append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	client := openai.NewClient(options...)

	return &OpenAIProvider{
		client: client,
	}
}

// Chat implements the LLM interface
func (p *OpenAIProvider) Chat(ctx context.Context, messages []llm.Message, opts ...llm.Option) (llm.Response, error) {
	options := llm.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Convert messages
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		openAIMsg, err := convertToOpenAIMessage(msg)
		if err != nil {
			return llm.Response{}, err
		}
		openAIMessages = append(openAIMessages, openAIMsg)
	}

	// Prepare params
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	// Set the model
	if options.Model != "" {
		params.Model = options.Model
	}

	// Set optional parameters
	if options.Temperature != 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}

	if options.TopP != 0 {
		params.TopP = openai.Float(float64(options.TopP))
	}

	if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}

	// Handle stop sequences
	if len(options.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfChatCompletionNewsStopArray: options.Stop,
		}
	}

	// Convert tools
	if len(options.Tools) > 0 || len(options.Functions) > 0 {
		tools := convertToOpenAITools(options.Tools, options.Functions)
		if len(tools) > 0 {
			params.Tools = tools
		}
	}

	// Set tool choice if specified
	if options.ToolChoice != nil {
		params.ToolChoice = convertToOpenAIToolChoice(options.ToolChoice)
	}

	// Set JSON mode if specified
	if options.JSONMode {
		params.ResponseFormat = convertToJSONFormatParam()
	} else if options.ResponseFormat != nil {
		params.ResponseFormat = convertToResponseFormatParam(options.ResponseFormat)
	}

	// Make the API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return llm.Response{}, err
	}

	// Convert the response
	return convertFromOpenAIResponse(completion)
}

// ChatStream implements the LLM interface for streaming responses
func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []llm.Message, opts ...llm.Option) (llm.Stream, error) {
	options := llm.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Convert messages
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		openAIMsg, err := convertToOpenAIMessage(msg)
		if err != nil {
			return nil, err
		}
		openAIMessages = append(openAIMessages, openAIMsg)
	}

	// Prepare params - note we don't set stream here, the NewStreaming method handles that
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	// Set the model
	if options.Model != "" {
		params.Model = options.Model
	}

	// Set optional parameters with type conversions
	if options.Temperature != 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}

	if options.TopP != 0 {
		params.TopP = openai.Float(float64(options.TopP))
	}

	if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}

	// Handle stop sequences
	if len(options.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfChatCompletionNewsStopArray: options.Stop,
		}
	}

	// Convert tools
	if len(options.Tools) > 0 || len(options.Functions) > 0 {
		tools := convertToOpenAITools(options.Tools, options.Functions)
		if len(tools) > 0 {
			params.Tools = tools
		}
	}

	// Set tool choice if specified
	if options.ToolChoice != nil {
		params.ToolChoice = convertToOpenAIToolChoice(options.ToolChoice)
	}

	// Set JSON mode if specified
	if options.JSONMode {
		params.ResponseFormat = convertToJSONFormatParam()
	} else if options.ResponseFormat != nil {
		params.ResponseFormat = convertToResponseFormatParam(options.ResponseFormat)
	}

	// Create the stream
	sseStream := p.client.Chat.Completions.NewStreaming(ctx, params)

	// Return our stream adapter
	return &openAIStream{
		stream:      sseStream,
		accumulator: openai.ChatCompletionAccumulator{},
	}, nil
}

// openAIStream adapts the OpenAI streaming response to our Stream interface
type openAIStream struct {
	stream      *ssestream.Stream[openai.ChatCompletionChunk]
	accumulator openai.ChatCompletionAccumulator
	lastError   error
	current     llm.Message
}

func (s *openAIStream) Next() (llm.Message, error) {
	// If we already encountered an error, return it
	if s.lastError != nil {
		return llm.Message{}, s.lastError
	}

	// Get the next event
	if !s.stream.Next() {
		if err := s.stream.Err(); err != nil {
			s.lastError = err
			return llm.Message{}, err
		}
		s.lastError = io.EOF
		return llm.Message{}, io.EOF
	}

	// Get the current chunk
	chunk := s.stream.Current()
	s.accumulator.AddChunk(chunk)

	if len(chunk.Choices) == 0 {
		return llm.Message{}, nil
	}

	delta := chunk.Choices[0].Delta

	// Update the current message
	s.current.Role = llm.RoleAssistant
	s.current.Content += delta.Content

	// Handle tool calls
	if len(delta.ToolCalls) > 0 {
		if s.current.ToolCalls == nil {
			s.current.ToolCalls = make([]llm.ToolCall, 0)
		}

		for _, tc := range delta.ToolCalls {
			// Find or create tool call
			found := false
			for i, existingTC := range s.current.ToolCalls {
				if existingTC.ID == tc.ID {
					// Update existing tool call
					s.current.ToolCalls[i].Function.Name += tc.Function.Name
					s.current.ToolCalls[i].Function.Arguments += tc.Function.Arguments
					found = true
					break
				}
			}

			if !found && tc.ID != "" {
				// Add new tool call
				s.current.ToolCalls = append(s.current.ToolCalls, llm.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
	}

	return s.current, nil
}

func (s *openAIStream) Close() error {
	// No explicit close method on the OpenAI stream
	return nil
}

// Helper functions to convert between the two libraries

func convertToOpenAIMessage(msg llm.Message) (openai.ChatCompletionMessageParamUnion, error) {
	switch msg.Role {
	case llm.RoleSystem:
		return openai.SystemMessage(msg.Content), nil
	case llm.RoleUser:
		return openai.UserMessage(msg.Content), nil
	case llm.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
					ID:   tc.ID,
					Type: "function",
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}

			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
					ToolCalls: toolCalls,
				},
			}, nil
		}

		return openai.AssistantMessage(msg.Content), nil
	case llm.RoleFunction:
		// Use the tool message approach for function messages
		return openai.ChatCompletionMessageParamUnion{
			OfTool: &openai.ChatCompletionToolMessageParam{
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				},
				ToolCallID: msg.Name, // Using name for tool call ID
			},
		}, nil
	case llm.RoleTool:
		return openai.ToolMessage(msg.Content, msg.ToolCallID), nil
	default:
		return openai.ChatCompletionMessageParamUnion{}, errors.New("unsupported role: " + msg.Role)
	}
}

func convertToOpenAITools(tools []llm.Tool, functions []llm.Function) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, 0)

	// Convert tools
	for _, tool := range tools {
		if tool.Type == "function" {
			// Create JSON parameters
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			var parametersMap map[string]interface{}
			_ = json.Unmarshal(paramsJSON, &parametersMap)

			result = append(result, openai.ChatCompletionToolParam{
				Type: constant.Function("function"),
				Function: shared.FunctionDefinitionParam{
					Name:        tool.Function.Name,
					Description: openai.String(tool.Function.Description),
					Parameters:  shared.FunctionParameters(parametersMap),
				},
			})
		}
	}

	// Convert legacy functions
	for _, fn := range functions {
		// Create JSON parameters
		paramsJSON, _ := json.Marshal(fn.Parameters)
		var parametersMap map[string]interface{}
		_ = json.Unmarshal(paramsJSON, &parametersMap)

		result = append(result, openai.ChatCompletionToolParam{
			Type: constant.Function("function"),
			Function: shared.FunctionDefinitionParam{
				Name:        fn.Name,
				Description: openai.String(fn.Description),
				Parameters:  shared.FunctionParameters(parametersMap),
			},
		})
	}

	return result
}

func convertToOpenAIToolChoice(toolChoice interface{}) openai.ChatCompletionToolChoiceOptionUnionParam {
	// Handle string choices like "auto" or "none"
	if strChoice, ok := toolChoice.(string); ok {
		if strChoice == "auto" {
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("auto"),
			}
		} else if strChoice == "none" {
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("none"),
			}
		}
	}

	// Handle map for specific function choice
	if mapChoice, ok := toolChoice.(map[string]interface{}); ok {
		if funcNameMap, ok := mapChoice["function"].(map[string]interface{}); ok {
			if name, ok := funcNameMap["name"].(string); ok {
				return openai.ChatCompletionToolChoiceOptionUnionParam{
					OfChatCompletionNamedToolChoice: &openai.ChatCompletionNamedToolChoiceParam{
						Type: "function",
						Function: openai.ChatCompletionNamedToolChoiceFunctionParam{
							Name: name,
						},
					},
				}
			}
		}
	}

	// Default to auto
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: openai.String("auto"),
	}
}

func convertToJSONFormatParam() openai.ChatCompletionNewParamsResponseFormatUnion {
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
	}
}

func convertToResponseFormatParam(format *llm.ResponseFormat) openai.ChatCompletionNewParamsResponseFormatUnion {
	switch format.Type {
	case llm.JSONObject:
		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	case llm.JSONSchema:
		schema, ok := format.JSONSchema.(map[string]interface{})
		if !ok {
			// Try to convert to map if it's not already
			schemaBytes, _ := json.Marshal(format.JSONSchema)
			var schemaMap map[string]interface{}
			if err := json.Unmarshal(schemaBytes, &schemaMap); err == nil {
				schema = schemaMap
			} else {
				return openai.ChatCompletionNewParamsResponseFormatUnion{}
			}
		}

		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "schema",
					Schema: schema,
				},
			},
		}
	default:
		// Default to text format
		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: &shared.ResponseFormatTextParam{},
		}
	}
}

func convertFromOpenAIResponse(completion *openai.ChatCompletion) (llm.Response, error) {
	if len(completion.Choices) == 0 {
		return llm.Response{}, errors.New("no choices in response")
	}

	// Get the first choice
	choice := completion.Choices[0]

	// Convert the message
	message := llm.Message{
		Role:    string(choice.Message.Role),
		Content: choice.Message.Content,
	}

	// Handle tool calls
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]llm.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, llm.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: llm.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		message.ToolCalls = toolCalls
	}

	// Convert usage
	usage := llm.Usage{
		PromptTokens:     int(completion.Usage.PromptTokens),
		CompletionTokens: int(completion.Usage.CompletionTokens),
		TotalTokens:      int(completion.Usage.TotalTokens),
	}

	return llm.Response{
		Message: message,
		Usage:   usage,
	}, nil
}
