package aiopenai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/ai/embedding"
	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/ocr"
	"github.com/Abraxas-365/craftable/ai/speech"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

// OpenAIProvider implements the LLM interface for OpenAI using Responses API
type OpenAIProvider struct {
	client     openai.Client
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, opts ...option.RequestOption) *OpenAIProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	options := append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	client := openai.NewClient(options...)

	return &OpenAIProvider{
		client:     client,
		httpClient: &http.Client{},
		apiKey:     apiKey,
		baseURL:    "https://api.openai.com/v1",
	}
}

func defaultChatOptions() *llm.ChatOptions {
	options := llm.DefaultOptions()
	options.Model = "gpt-4o"
	options.UseResponsesAPI = true // Default to Responses API
	return options
}

// ResponsesInput represents input messages for the Responses API
type ResponsesInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponsesTool represents a tool in the Responses API
type ResponsesTool struct {
	Type     string                 `json:"type"`
	Function *ResponsesToolFunction `json:"function,omitempty"`
}

type ResponsesToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ResponsesRequest represents a request to the Responses API
type ResponsesRequest struct {
	Model               string              `json:"model"`
	Input               []ResponsesInput    `json:"input"`
	Tools               []ResponsesTool     `json:"tools,omitempty"`
	PreviousResponseID  string              `json:"previous_response_id,omitempty"`
	Temperature         *float64            `json:"temperature,omitempty"`
	MaxCompletionTokens *int                `json:"max_completion_tokens,omitempty"`
	TopP                *float64            `json:"top_p,omitempty"`
	PresencePenalty     *float64            `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float64            `json:"frequency_penalty,omitempty"`
	Seed                *int64              `json:"seed,omitempty"`
	User                string              `json:"user,omitempty"`
	Stream              bool                `json:"stream,omitempty"`
	ResponseFormat      map[string]any      `json:"response_format,omitempty"`
	Reasoning           *ResponsesReasoning `json:"reasoning,omitempty"`
	Store               bool                `json:"store,omitempty"`
	Metadata            map[string]string   `json:"metadata,omitempty"`
}

type ResponsesReasoning struct {
	Effort string `json:"effort,omitempty"` // "low", "medium", "high"
}

// ResponsesChoice represents a choice in the response
type ResponsesChoice struct {
	Index        int              `json:"index"`
	Message      ResponsesMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type ResponsesMessage struct {
	Role      string              `json:"role"`
	Content   string              `json:"content"`
	ToolCalls []ResponsesToolCall `json:"tool_calls,omitempty"`
}

type ResponsesToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ResponsesResponse represents a response from the Responses API
type ResponsesResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []ResponsesChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat implements the LLM interface using Responses API
func (p *OpenAIProvider) Chat(ctx context.Context, messages []llm.Message, opts ...llm.Option) (llm.Response, error) {
	options := defaultChatOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Use Responses API if specified (default) or Chat Completions as fallback
	if options.UseResponsesAPI {
		return p.chatWithResponses(ctx, messages, options)
	}
	return p.chatWithCompletions(ctx, messages, options)
}

// chatWithResponses uses the Responses API
func (p *OpenAIProvider) chatWithResponses(ctx context.Context, messages []llm.Message, options *llm.ChatOptions) (llm.Response, error) {
	// Convert messages to Responses API format
	input := make([]ResponsesInput, 0, len(messages))
	for _, msg := range messages {
		role := msg.Role
		// Convert "system" to "developer" for Responses API
		if role == llm.RoleSystem {
			role = "developer"
		}
		input = append(input, ResponsesInput{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Convert tools if any
	var tools []ResponsesTool
	if len(options.Tools) > 0 {
		tools = make([]ResponsesTool, 0, len(options.Tools))
		for _, tool := range options.Tools {
			if tool.Type == "function" {
				// Convert parameters to map[string]any
				var params map[string]any
				if tool.Function.Parameters != nil {
					switch p := tool.Function.Parameters.(type) {
					case map[string]any:
						params = p
					default:
						// Try to marshal and unmarshal to convert
						paramBytes, _ := json.Marshal(tool.Function.Parameters)
						_ = json.Unmarshal(paramBytes, &params)
					}
				}

				tools = append(tools, ResponsesTool{
					Type: "function",
					Function: &ResponsesToolFunction{
						Name:        tool.Function.Name,
						Description: tool.Function.Description,
						Parameters:  params,
					},
				})
			} else {
				// Built-in tools like "web_search", "file_search"
				tools = append(tools, ResponsesTool{Type: tool.Type})
			}
		}
	}

	// Build request
	req := ResponsesRequest{
		Model: options.Model,
		Input: input,
		Tools: tools,
		Store: true, // Enable state storage by default
	}

	// Set optional parameters
	if options.Temperature != 0 {
		temp := float64(options.Temperature)
		req.Temperature = &temp
	}

	if options.TopP != 0 {
		topP := float64(options.TopP)
		req.TopP = &topP
	}

	if options.MaxCompletionTokens > 0 {
		req.MaxCompletionTokens = &options.MaxCompletionTokens
	} else if options.MaxTokens > 0 {
		// Fallback to legacy MaxTokens
		req.MaxCompletionTokens = &options.MaxTokens
	}

	if options.PresencePenalty != 0 {
		penalty := float64(options.PresencePenalty)
		req.PresencePenalty = &penalty
	}

	if options.FrequencyPenalty != 0 {
		penalty := float64(options.FrequencyPenalty)
		req.FrequencyPenalty = &penalty
	}

	if options.Seed != 0 {
		req.Seed = &options.Seed
	}

	if options.User != "" {
		req.User = options.User
	}

	// Handle reasoning effort for reasoning models
	if options.ReasoningEffort != "" {
		req.Reasoning = &ResponsesReasoning{
			Effort: options.ReasoningEffort,
		}
	}

	// Handle response format
	if options.JSONMode {
		req.ResponseFormat = map[string]any{
			"type": "json_object",
		}
	} else if options.ResponseFormat != nil {
		switch options.ResponseFormat.Type {
		case llm.JSONObject:
			req.ResponseFormat = map[string]any{
				"type": "json_object",
			}
		case llm.JSONSchema:
			req.ResponseFormat = map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "schema",
					"schema": options.ResponseFormat.JSONSchema,
				},
			}
		}
	}

	// Make the API call
	apiResp, err := p.callResponsesAPI(ctx, req, options.Headers)
	if err != nil {
		return llm.Response{}, err
	}

	// Convert to llm.Response
	if len(apiResp.Choices) == 0 {
		return llm.Response{}, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	message := llm.Message{
		Role:    choice.Message.Role,
		Content: choice.Message.Content,
	}

	// Convert tool calls if any
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]llm.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, llm.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: llm.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		message.ToolCalls = toolCalls
	}

	return llm.Response{
		Message: message,
		Usage: llm.Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}, nil
}

// callResponsesAPI makes a direct HTTP call to the Responses API
func (p *OpenAIProvider) callResponsesAPI(ctx context.Context, req ResponsesRequest, headers map[string]string) (*ResponsesResponse, error) {
	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/responses", p.baseURL),
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Add custom headers
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// Make request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var apiResp ResponsesResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResp, nil
}

// chatWithCompletions uses traditional Chat Completions API as fallback
func (p *OpenAIProvider) chatWithCompletions(ctx context.Context, messages []llm.Message, options *llm.ChatOptions) (llm.Response, error) {
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
		Model:    options.Model,
	}

	// Set optional parameters
	if options.Temperature != 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}

	if options.TopP != 0 {
		params.TopP = openai.Float(float64(options.TopP))
	}

	if options.MaxCompletionTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(options.MaxCompletionTokens))
	} else if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}

	if options.PresencePenalty != 0 {
		params.PresencePenalty = openai.Float(float64(options.PresencePenalty))
	}

	if options.FrequencyPenalty != 0 {
		params.FrequencyPenalty = openai.Float(float64(options.FrequencyPenalty))
	}

	if len(options.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfChatCompletionNewsStopArray: options.Stop,
		}
	}

	if options.Seed != 0 {
		params.Seed = openai.Int(options.Seed)
	}

	if options.User != "" {
		params.User = openai.String(options.User)
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

// ChatStream implements streaming for both APIs
func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []llm.Message, opts ...llm.Option) (llm.Stream, error) {
	options := defaultChatOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Responses API streaming would go here when SDK supports it
	// For now, use Chat Completions streaming
	return p.chatStreamWithCompletions(ctx, messages, options)
}

// chatStreamWithCompletions uses Chat Completions API for streaming
func (p *OpenAIProvider) chatStreamWithCompletions(ctx context.Context, messages []llm.Message, options *llm.ChatOptions) (llm.Stream, error) {
	// Convert messages
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		openAIMsg, err := convertToOpenAIMessage(msg)
		if err != nil {
			return nil, err
		}
		openAIMessages = append(openAIMessages, openAIMsg)
	}

	// Prepare params
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
		Model:    options.Model,
	}

	// Set optional parameters
	if options.Temperature != 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}

	if options.TopP != 0 {
		params.TopP = openai.Float(float64(options.TopP))
	}

	if options.MaxCompletionTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(options.MaxCompletionTokens))
	} else if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}

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
	stream interface {
		Next() bool
		Current() openai.ChatCompletionChunk
		Err() error
	}
	accumulator openai.ChatCompletionAccumulator
	lastError   error
	current     llm.Message
}

func (s *openAIStream) Next() (llm.Message, error) {
	if s.lastError != nil {
		return llm.Message{}, s.lastError
	}

	if !s.stream.Next() {
		if err := s.stream.Err(); err != nil {
			s.lastError = err
			return llm.Message{}, err
		}
		s.lastError = io.EOF
		return llm.Message{}, io.EOF
	}

	chunk := s.stream.Current()
	s.accumulator.AddChunk(chunk)

	if len(chunk.Choices) == 0 {
		return llm.Message{}, nil
	}

	delta := chunk.Choices[0].Delta

	s.current.Role = llm.RoleAssistant
	s.current.Content += delta.Content

	if len(delta.ToolCalls) > 0 {
		if s.current.ToolCalls == nil {
			s.current.ToolCalls = make([]llm.ToolCall, 0)
		}

		for _, tc := range delta.ToolCalls {
			found := false
			for i, existingTC := range s.current.ToolCalls {
				if existingTC.ID == tc.ID {
					s.current.ToolCalls[i].Function.Name += tc.Function.Name
					s.current.ToolCalls[i].Function.Arguments += tc.Function.Arguments
					found = true
					break
				}
			}

			if !found && tc.ID != "" {
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
	return nil
}

// Helper functions

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
		return openai.ChatCompletionMessageParamUnion{
			OfTool: &openai.ChatCompletionToolMessageParam{
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: openai.String(msg.Content),
				},
				ToolCallID: msg.Name,
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

	for _, tool := range tools {
		if tool.Type == "function" {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			var parametersMap map[string]any
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

	for _, fn := range functions {
		paramsJSON, _ := json.Marshal(fn.Parameters)
		var parametersMap map[string]any
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

func convertToOpenAIToolChoice(toolChoice any) openai.ChatCompletionToolChoiceOptionUnionParam {
	if strChoice, ok := toolChoice.(string); ok {
		if strChoice == "auto" {
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("auto"),
			}
		} else if strChoice == "none" {
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("none"),
			}
		} else if strChoice == "required" {
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("required"),
			}
		}
	}

	if mapChoice, ok := toolChoice.(map[string]any); ok {
		if funcNameMap, ok := mapChoice["function"].(map[string]any); ok {
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
		schema, ok := format.JSONSchema.(map[string]any)
		if !ok {
			schemaBytes, _ := json.Marshal(format.JSONSchema)
			var schemaMap map[string]any
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
		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: &shared.ResponseFormatTextParam{},
		}
	}
}

func convertFromOpenAIResponse(completion *openai.ChatCompletion) (llm.Response, error) {
	if len(completion.Choices) == 0 {
		return llm.Response{}, errors.New("no choices in response")
	}

	choice := completion.Choices[0]

	message := llm.Message{
		Role:    string(choice.Message.Role),
		Content: choice.Message.Content,
	}

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

func (p *OpenAIProvider) EmbedDocuments(ctx context.Context, documents []string, opts ...embedding.Option) ([]embedding.Embedding, error) {
	options := embedding.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: documents,
		},
	}

	if options.Model != "" {
		params.Model = options.Model
	} else {
		params.Model = "text-embedding-3-small"
	}

	if options.Dimensions > 0 {
		params.Dimensions = openai.Int(int64(options.Dimensions))
	}

	if options.User != "" {
		params.User = openai.String(options.User)
	}

	resp, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, err
	}

	embeddings := make([]embedding.Embedding, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = embedding.Embedding{
			Vector: convertToFloat32Slice(data.Embedding),
			Usage: embedding.Usage{
				PromptTokens: int(resp.Usage.PromptTokens),
				TotalTokens:  int(resp.Usage.TotalTokens),
			},
		}
	}

	return embeddings, nil
}

func (p *OpenAIProvider) EmbedQuery(ctx context.Context, text string, opts ...embedding.Option) (embedding.Embedding, error) {
	embeddings, err := p.EmbedDocuments(ctx, []string{text}, opts...)
	if err != nil {
		return embedding.Embedding{}, err
	}

	if len(embeddings) == 0 {
		return embedding.Embedding{}, errors.New("no embedding returned")
	}

	return embeddings[0], nil
}

func convertToFloat32Slice(input []float64) []float32 {
	result := make([]float32, len(input))
	for i, v := range input {
		result[i] = float32(v)
	}
	return result
}

func (p *OpenAIProvider) ExtractText(ctx context.Context, imageData []byte, opts ...ocr.Option) (ocr.Result, error) {
	options := ocr.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	systemContent := "You are an OCR system that extracts text from images. "
	if options.Language != "auto" {
		systemContent += fmt.Sprintf("The text is in %s. ", options.Language)
	}
	if options.DetectOrientation {
		systemContent += "Detect and account for text orientation. "
	}

	userContent := "Extract the text from this image. "

	switch options.DetailsLevel {
	case "high":
		userContent += "Provide the text, confidence level, and approximate positions of text blocks."
	case "medium":
		userContent += "Provide the text and confidence level."
	case "low":
		userContent += "Just provide the extracted text, nothing else."
	}

	openAIMessages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemContent),
	}

	contentParts := []openai.ChatCompletionContentPartUnionParam{
		{
			OfText: &openai.ChatCompletionContentPartTextParam{
				Type: constant.Text("text"),
				Text: userContent,
			},
		},
		{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				Type: constant.ImageURL("image_url"),
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
					Detail: "high",
				},
			},
		},
	}

	openAIMessages = append(openAIMessages, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: contentParts,
			},
		},
	})

	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	modelToUse := options.Model
	if modelToUse == "" {
		modelToUse = "gpt-4-vision-preview"
	}
	params.Model = modelToUse
	params.MaxTokens = openai.Int(1024)

	if options.User != "" {
		params.User = openai.String(options.User)
	}

	startTime := time.Now()

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ocr.Result{}, err
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	if len(completion.Choices) == 0 {
		return ocr.Result{}, errors.New("no response from API")
	}

	textContent := completion.Choices[0].Message.Content

	result := ocr.Result{
		Text: textContent,
		Usage: ocr.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
			ProcessingTime:   processingTime,
		},
	}

	if options.DetailsLevel != "low" {
		result.Confidence = estimateConfidence(textContent)

		if options.DetailsLevel == "high" {
			result.Blocks = parseTextBlocks(textContent)
		}
	}

	return result, nil
}

func (p *OpenAIProvider) ExtractTextFromURL(ctx context.Context, imageURL string, opts ...ocr.Option) (ocr.Result, error) {
	options := ocr.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	systemContent := "You are an OCR system that extracts text from images. "
	if options.Language != "auto" {
		systemContent += fmt.Sprintf("The text is in %s. ", options.Language)
	}
	if options.DetectOrientation {
		systemContent += "Detect and account for text orientation. "
	}

	userContent := "Extract the text from this image. "

	switch options.DetailsLevel {
	case "high":
		userContent += "Provide the text, confidence level, and approximate positions of text blocks."
	case "medium":
		userContent += "Provide the text and confidence level."
	case "low":
		userContent += "Just provide the extracted text, nothing else."
	}

	openAIMessages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemContent),
	}

	contentParts := []openai.ChatCompletionContentPartUnionParam{
		{
			OfText: &openai.ChatCompletionContentPartTextParam{
				Type: constant.Text("text"),
				Text: userContent,
			},
		},
		{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				Type: constant.ImageURL("image_url"),
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    imageURL,
					Detail: "high",
				},
			},
		},
	}

	openAIMessages = append(openAIMessages, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: contentParts,
			},
		},
	})

	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	modelToUse := options.Model
	if modelToUse == "" {
		modelToUse = "gpt-4-vision-preview"
	}
	params.Model = modelToUse
	params.MaxTokens = openai.Int(1024)

	if options.User != "" {
		params.User = openai.String(options.User)
	}

	startTime := time.Now()

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ocr.Result{}, err
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	if len(completion.Choices) == 0 {
		return ocr.Result{}, errors.New("no response from API")
	}

	textContent := completion.Choices[0].Message.Content

	result := ocr.Result{
		Text: textContent,
		Usage: ocr.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
			ProcessingTime:   processingTime,
		},
	}

	if options.DetailsLevel != "low" {
		result.Confidence = estimateConfidence(textContent)

		if options.DetailsLevel == "high" {
			result.Blocks = parseTextBlocks(textContent)
		}
	}

	return result, nil
}

func estimateConfidence(text string) float32 {
	if strings.Contains(strings.ToLower(text), "low confidence") {
		return 0.3
	} else if strings.Contains(strings.ToLower(text), "medium confidence") {
		return 0.6
	} else if strings.Contains(strings.ToLower(text), "high confidence") {
		return 0.9
	}
	return 0.7
}

func parseTextBlocks(text string) []ocr.TextBlock {
	blocks := []ocr.TextBlock{}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			blocks = append(blocks, ocr.TextBlock{
				Text:       line,
				Confidence: 0.7,
				BoundingBox: ocr.BoundingBox{
					Y:      float32(i) / float32(len(lines)),
					X:      0.1,
					Width:  0.8,
					Height: 1.0 / float32(len(lines)),
				},
			})
		}
	}

	return blocks
}

func (p *OpenAIProvider) Synthesize(ctx context.Context, text string, opts ...speech.SynthesisOption) (speech.Audio, error) {
	options := speech.SynthesisOptions{
		Model:       string(openai.SpeechModelTTS1),
		Voice:       "alloy",
		AudioFormat: speech.AudioFormatMP3,
		SpeechRate:  1.0,
	}

	for _, opt := range opts {
		opt(&options)
	}

	responseFormat := openai.AudioSpeechNewParamsResponseFormatMP3
	switch options.AudioFormat {
	case speech.AudioFormatMP3:
		responseFormat = openai.AudioSpeechNewParamsResponseFormatMP3
	case speech.AudioFormatPCM:
		responseFormat = openai.AudioSpeechNewParamsResponseFormatPCM
	case speech.AudioFormatOGG:
		responseFormat = openai.AudioSpeechNewParamsResponseFormatOpus
	case speech.AudioFormatWAV:
		responseFormat = openai.AudioSpeechNewParamsResponseFormatPCM
	}

	voice := openai.AudioSpeechNewParamsVoiceAlloy
	switch strings.ToLower(options.Voice) {
	case "alloy":
		voice = openai.AudioSpeechNewParamsVoiceAlloy
	case "echo":
		voice = openai.AudioSpeechNewParamsVoiceEcho
	case "fable":
		voice = openai.AudioSpeechNewParamsVoiceFable
	case "onyx":
		voice = openai.AudioSpeechNewParamsVoiceOnyx
	case "nova":
		voice = openai.AudioSpeechNewParamsVoiceNova
	case "shimmer":
		voice = openai.AudioSpeechNewParamsVoiceShimmer
	}

	params := openai.AudioSpeechNewParams{
		Model:          options.Model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: responseFormat,
	}

	if options.SpeechRate != 1.0 {
		params.Speed = param.NewOpt(float64(options.SpeechRate))
	}

	res, err := p.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return speech.Audio{}, fmt.Errorf("openai speech synthesis error: %w", err)
	}

	sampleRate := 24000
	if options.SampleRate > 0 {
		sampleRate = options.SampleRate
	}

	return speech.Audio{
		Content:    res.Body,
		Format:     options.AudioFormat,
		SampleRate: sampleRate,
		Usage: speech.TTSUsage{
			InputCharacters: len(text),
		},
	}, nil
}

func (p *OpenAIProvider) Transcribe(ctx context.Context, audio io.Reader, opts ...speech.TranscriptionOption) (speech.Transcript, error) {
	options := speech.TranscriptionOptions{
		Model:      string(openai.AudioModelWhisper1),
		Language:   "",
		Timestamps: false,
	}

	for _, opt := range opts {
		opt(&options)
	}

	params := openai.AudioTranscriptionNewParams{
		Model: options.Model,
		File:  audio,
	}

	if options.Language != "" {
		params.Language = param.NewOpt(options.Language)
	}

	response, err := p.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return speech.Transcript{}, fmt.Errorf("openai transcription error: %w", err)
	}

	result := speech.Transcript{
		Text: response.Text,
	}

	return result, nil
}

