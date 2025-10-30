package aiopenai

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
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
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
			OfStringArray: options.Stop,
		}
	}

	if options.Seed != 0 {
		params.Seed = openai.Int(options.Seed)
	}

	if options.User != "" {
		params.User = openai.String(options.User)
	}

	// Add LogitBias
	if len(options.LogitBias) > 0 {
		logitBias := make(map[string]int64)
		for k, v := range options.LogitBias {
			logitBias[fmt.Sprintf("%d", k)] = int64(v)
		}
		params.LogitBias = logitBias
	}

	// Set reasoning effort for reasoning models (o1, o3, etc.)
	if options.ReasoningEffort != "" {
		params.ReasoningEffort = convertToOpenAIReasoningEffort(options.ReasoningEffort)
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

// ChatStream implements streaming for Chat Completions API
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
			OfStringArray: options.Stop,
		}
	}

	// Set reasoning effort for reasoning models
	if options.ReasoningEffort != "" {
		params.ReasoningEffort = convertToOpenAIReasoningEffort(options.ReasoningEffort)
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
			// Build tool calls using the correct Param types
			toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID:   tc.ID,
						Type: constant.Function("function"),
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					},
				})
			}

			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Role: constant.Assistant("assistant"),
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

func convertToOpenAITools(tools []llm.Tool, functions []llm.Function) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, 0)

	for _, tool := range tools {
		if tool.Type == "function" {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			var parametersMap map[string]any
			_ = json.Unmarshal(paramsJSON, &parametersMap)

			result = append(result, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: openai.String(tool.Function.Description),
				Parameters:  openai.FunctionParameters(parametersMap),
			}))
		}
	}

	for _, fn := range functions {
		paramsJSON, _ := json.Marshal(fn.Parameters)
		var parametersMap map[string]any
		_ = json.Unmarshal(paramsJSON, &parametersMap)

		result = append(result, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        fn.Name,
			Description: openai.String(fn.Description),
			Parameters:  openai.FunctionParameters(parametersMap),
		}))
	}

	return result
}

func convertToOpenAIToolChoice(toolChoice any) openai.ChatCompletionToolChoiceOptionUnionParam {
	if strChoice, ok := toolChoice.(string); ok {
		switch strChoice {
		case "auto":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("auto"),
			}
		case "none":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("none"),
			}
		case "required":
			return openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("required"),
			}
		}
	}

	// For specific tool choice - simplified approach
	// Just default to auto if we can't parse it properly
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: openai.String("auto"),
	}
}

func convertToOpenAIReasoningEffort(effort string) shared.ReasoningEffort {
	switch strings.ToLower(effort) {
	case "low":
		return shared.ReasoningEffortLow
	case "medium":
		return shared.ReasoningEffortMedium
	case "high":
		return shared.ReasoningEffortHigh
	case "minimal":
		return shared.ReasoningEffortMinimal
	default:
		// Default to medium if unrecognized
		return shared.ReasoningEffortMedium
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
	default:
		voice = openai.AudioSpeechNewParamsVoiceAlloy
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

