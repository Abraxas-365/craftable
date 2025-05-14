package aiproviders

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func defaultChatOptions() *llm.ChatOptions {
	options := llm.DefaultOptions()
	options.Model = "gpt-4o"
	return options
}

// Chat implements the LLM interface
func (p *OpenAIProvider) Chat(ctx context.Context, messages []llm.Message, opts ...llm.Option) (llm.Response, error) {
	options := defaultChatOptions()
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
	options := defaultChatOptions()
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

func (p *OpenAIProvider) EmbedDocuments(ctx context.Context, documents []string, opts ...embedding.Option) ([]embedding.Embedding, error) {
	options := embedding.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Prepare parameters
	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: documents,
		},
	}

	// Set the model
	if options.Model != "" {
		params.Model = options.Model
	} else {
		// Default model
		params.Model = "text-embedding-3-small"
	}

	// Set dimensions if specified
	if options.Dimensions > 0 {
		params.Dimensions = openai.Int(int64(options.Dimensions))
	}

	// Set user if specified
	if options.User != "" {
		params.User = openai.String(options.User)
	}

	// Call the OpenAI embedding API
	resp, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert the response to our embedding type
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

// EmbedQuery implements the embedding.Embedder interface
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

// Helper function to convert []float64 to []float32
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

	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Create system message with instructions based on options
	systemContent := "You are an OCR system that extracts text from images. "
	if options.Language != "auto" {
		systemContent += fmt.Sprintf("The text is in %s. ", options.Language)
	}
	if options.DetectOrientation {
		systemContent += "Detect and account for text orientation. "
	}

	// Construct the prompt for the vision model
	userContent := "Extract the text from this image. "

	switch options.DetailsLevel {
	case "high":
		userContent += "Provide the text, confidence level, and approximate positions of text blocks."
	case "medium":
		userContent += "Provide the text and confidence level."
	case "low":
		userContent += "Just provide the extracted text, nothing else."
	}

	// Create the OpenAI API request parameters
	openAIMessages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemContent),
	}

	// Create content parts for the user message
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
					Detail: "high", // Use high detail for better OCR
				},
			},
		},
	}

	// Create the user message with content parts
	openAIMessages = append(openAIMessages, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: contentParts,
			},
		},
	})

	// Prepare params with the messages
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	// Set the model
	modelToUse := options.Model
	if modelToUse == "" {
		modelToUse = "gpt-4-vision-preview"
	}
	params.Model = modelToUse

	// Set max tokens - vision models often need higher limits
	params.MaxTokens = openai.Int(1024)

	// Set user if specified
	if options.User != "" {
		params.User = openai.String(options.User)
	}

	// Track time for processing
	startTime := time.Now()

	// Call the OpenAI API directly
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ocr.Result{}, err
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	// Get the response content
	if len(completion.Choices) == 0 {
		return ocr.Result{}, errors.New("no response from API")
	}

	textContent := completion.Choices[0].Message.Content

	// Create the result
	result := ocr.Result{
		Text: textContent,
		Usage: ocr.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
			ProcessingTime:   processingTime,
		},
	}

	// For higher detail levels, attempt to parse structure from the response
	if options.DetailsLevel != "low" {
		// Estimate confidence
		result.Confidence = estimateConfidence(textContent)

		// For high detail, attempt to parse text blocks
		if options.DetailsLevel == "high" {
			result.Blocks = parseTextBlocks(textContent)
		}
	}

	return result, nil
}

// ExtractTextFromURL implements the ocr.OCRProvider interface
func (p *OpenAIProvider) ExtractTextFromURL(ctx context.Context, imageURL string, opts ...ocr.Option) (ocr.Result, error) {
	options := ocr.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create system message with instructions based on options
	systemContent := "You are an OCR system that extracts text from images. "
	if options.Language != "auto" {
		systemContent += fmt.Sprintf("The text is in %s. ", options.Language)
	}
	if options.DetectOrientation {
		systemContent += "Detect and account for text orientation. "
	}

	// Construct the prompt for the vision model
	userContent := "Extract the text from this image. "

	switch options.DetailsLevel {
	case "high":
		userContent += "Provide the text, confidence level, and approximate positions of text blocks."
	case "medium":
		userContent += "Provide the text and confidence level."
	case "low":
		userContent += "Just provide the extracted text, nothing else."
	}

	// Create the OpenAI API request parameters
	openAIMessages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemContent),
	}

	// Create content parts for the user message
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
					Detail: "high", // Use high detail for better OCR
				},
			},
		},
	}

	// Create the user message with content parts
	openAIMessages = append(openAIMessages, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: contentParts,
			},
		},
	})

	// Prepare params with the messages
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
	}

	// Set the model
	modelToUse := options.Model
	if modelToUse == "" {
		modelToUse = "gpt-4-vision-preview"
	}
	params.Model = modelToUse

	// Set max tokens - vision models often need higher limits
	params.MaxTokens = openai.Int(1024)

	// Set user if specified
	if options.User != "" {
		params.User = openai.String(options.User)
	}

	// Track time for processing
	startTime := time.Now()

	// Call the OpenAI API directly
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ocr.Result{}, err
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	// Get the response content
	if len(completion.Choices) == 0 {
		return ocr.Result{}, errors.New("no response from API")
	}

	textContent := completion.Choices[0].Message.Content

	// Create the result
	result := ocr.Result{
		Text: textContent,
		Usage: ocr.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
			ProcessingTime:   processingTime,
		},
	}

	// For higher detail levels, attempt to parse structure from the response
	if options.DetailsLevel != "low" {
		// Estimate confidence
		result.Confidence = estimateConfidence(textContent)

		// For high detail, attempt to parse text blocks
		if options.DetailsLevel == "high" {
			result.Blocks = parseTextBlocks(textContent)
		}
	}

	return result, nil
}

// Helper functions for OCR

// Helper function to estimate confidence from text response
func estimateConfidence(text string) float32 {
	// This is a simple heuristic - in a real implementation you would
	// parse the actual confidence values from the structured response
	if strings.Contains(strings.ToLower(text), "low confidence") {
		return 0.3
	} else if strings.Contains(strings.ToLower(text), "medium confidence") {
		return 0.6
	} else if strings.Contains(strings.ToLower(text), "high confidence") {
		return 0.9
	}
	return 0.7 // Default confidence
}

// Helper function to parse text blocks from response
func parseTextBlocks(text string) []ocr.TextBlock {
	// In a real implementation, you would parse the actual blocks
	// from a structured response. This is just a placeholder.
	blocks := []ocr.TextBlock{}

	// Simple parsing logic - split by lines and treat each as a block
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			blocks = append(blocks, ocr.TextBlock{
				Text:       line,
				Confidence: 0.7, // Default confidence
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
	// Apply options
	options := speech.SynthesisOptions{
		Model:       string(openai.SpeechModelTTS1),
		Voice:       "alloy",
		AudioFormat: speech.AudioFormatMP3,
		SpeechRate:  1.0,
	}

	for _, opt := range opts {
		opt(&options)
	}

	// Map our format to OpenAI's format
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

	// Map our voice to OpenAI's voice format
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

	// Create params with required fields
	params := openai.AudioSpeechNewParams{
		Model:          options.Model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: responseFormat,
	}

	// Add optional Speed parameter if specified
	if options.SpeechRate != 1.0 {
		params.Speed = param.NewOpt(float64(options.SpeechRate))
	}

	// Make the API call
	res, err := p.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return speech.Audio{}, fmt.Errorf("openai speech synthesis error: %w", err)
	}

	// Determine sample rate based on the model
	sampleRate := 24000 // Default for TTS-1
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

// Transcribe converts speech audio to text using OpenAI's transcription API
func (p *OpenAIProvider) Transcribe(ctx context.Context, audio io.Reader, opts ...speech.TranscriptionOption) (speech.Transcript, error) {
	// Apply options
	options := speech.TranscriptionOptions{
		Model:      string(openai.AudioModelWhisper1),
		Language:   "",
		Timestamps: false,
	}

	for _, opt := range opts {
		opt(&options)
	}

	// Prepare API call parameters with required fields
	params := openai.AudioTranscriptionNewParams{
		Model: options.Model,
		File:  audio,
	}

	// Add optional Language parameter if specified
	if options.Language != "" {
		params.Language = param.NewOpt(options.Language)
	}

	// Make the API call
	response, err := p.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return speech.Transcript{}, fmt.Errorf("openai transcription error: %w", err)
	}

	result := speech.Transcript{
		Text: response.Text,
	}

	return result, nil
}
