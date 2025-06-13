package agentx

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/craftable/ai/llm/toolx"
)

// Agent represents an LLM-powered agent with memory and tool capabilities
type Agent struct {
	client  *llm.Client
	tools   *toolx.ToolxClient
	memory  memoryx.Memory
	options []llm.Option
}

// AgentOption configures an Agent
type AgentOption func(*Agent)

// WithOptions adds LLM options to the agent
func WithOptions(options ...llm.Option) AgentOption {
	return func(a *Agent) {
		a.options = append(a.options, options...)
	}
}

// WithTools adds tools to the agent
func WithTools(tools *toolx.ToolxClient) AgentOption {
	return func(a *Agent) {
		a.tools = tools
	}
}

// New creates a new agent
func New(client llm.Client, memory memoryx.Memory, opts ...AgentOption) *Agent {
	agent := &Agent{
		client: &client,
		memory: memory,
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

// Run processes a user message and returns the final response
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	// Add user message to memory
	if err := a.memory.Add(llm.NewUserMessage(userInput)); err != nil {
		return "", fmt.Errorf("failed to add user message: %w", err)
	}

	// Get messages from memory
	messages, err := a.memory.Messages()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Check if tools are available and add them as options if so
	options := a.options
	if a.tools != nil {
		// Convert tools to LLM-compatible format
		toolList := a.getToolsList()
		if len(toolList) > 0 {
			options = append(options, llm.WithTools(toolList))
		}
	}

	// Get response from LLM
	response, err := a.client.Chat(ctx, messages, options...)
	if err != nil {
		return "", fmt.Errorf("LLM error: %w", err)
	}

	// Add the response to memory
	if err := a.memory.Add(response.Message); err != nil {
		return "", fmt.Errorf("failed to add assistant response: %w", err)
	}

	// Check if the response contains tool calls
	if len(response.Message.ToolCalls) > 0 && a.tools != nil {
		return a.handleToolCalls(ctx, response.Message.ToolCalls)
	}

	return response.Message.Content, nil
}

// RunStream streams the agent's initial response
// Note: This doesn't handle tool calls in streaming mode
func (a *Agent) RunStream(ctx context.Context, userInput string) (llm.Stream, error) {
	// Add user message to memory
	if err := a.memory.Add(llm.NewUserMessage(userInput)); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// Get messages from memory
	messages, err := a.memory.Messages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Check if tools are available and add them as options if so
	options := a.options
	if a.tools != nil {
		toolList := a.getToolsList()
		if len(toolList) > 0 {
			options = append(options, llm.WithTools(toolList))
		}
	}

	// Get streaming response
	return a.client.ChatStream(ctx, messages, options...)
}

// handleToolCalls processes tool calls and returns the final response
func (a *Agent) handleToolCalls(ctx context.Context, toolCalls []llm.ToolCall) (string, error) {
	// Process each tool call
	for _, tc := range toolCalls {
		// Call the tool
		toolResponse, err := a.tools.Call(ctx, tc)
		if err != nil {
			return "", fmt.Errorf("tool execution error: %w", err)
		}

		// Add tool response to memory
		if err := a.memory.Add(toolResponse); err != nil {
			return "", fmt.Errorf("failed to add tool response: %w", err)
		}
	}

	// Get messages from memory
	messages, err := a.memory.Messages()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Get next response from LLM with tool results
	response, err := a.client.Chat(ctx, messages, a.options...)
	if err != nil {
		return "", fmt.Errorf("LLM error: %w", err)
	}

	// Add the response to memory
	if err := a.memory.Add(response.Message); err != nil {
		return "", fmt.Errorf("failed to add assistant response: %w", err)
	}

	// Check if we have more tool calls to handle
	if len(response.Message.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, response.Message.ToolCalls)
	}

	return response.Message.Content, nil
}

// getToolsList converts the tools to LLM-compatible format
func (a *Agent) getToolsList() []llm.Tool {
	return a.tools.GetTools()
}

// ClearMemory resets the conversation but keeps the system prompt
func (a *Agent) ClearMemory() error {
	return a.memory.Clear()
}

// AddMessage adds a message to memory
func (a *Agent) AddMessage(message llm.Message) error {
	return a.memory.Add(message)
}

// Messages returns all messages in memory
func (a *Agent) Messages() ([]llm.Message, error) {
	return a.memory.Messages()
}

// StreamWithTools streams responses while handling tool calls
// This is a more advanced implementation that processes tool calls in streaming mode
func (a *Agent) StreamWithTools(ctx context.Context, userInput string, streamHandler func(chunk string)) error {
	if err := a.memory.Add(llm.NewUserMessage(userInput)); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	messages, err := a.memory.Messages()
	if err != nil {
		return fmt.Errorf("failed to retrieve messages: %w", err)
	}

	options := a.options
	if a.tools != nil {
		toolList := a.getToolsList()
		if len(toolList) > 0 {
			options = append(options, llm.WithTools(toolList))
		}
	}

	// Initial streaming response
	stream, err := a.client.ChatStream(ctx, messages, options...)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Collect the full message and stream chunks to the handler
	var fullMessage llm.Message
	var responseContent string
	var toolCalls []llm.ToolCall

	for {
		chunk, err := stream.Next()
		if err != nil {
			// Check if it's the end of the stream
			if errors.Is(err, io.EOF) {
				// Some implementations might return a final chunk with the error
				if chunk.Role != "" {
					fullMessage = chunk
				}
				break
			}
			// Any other error is returned
			return err
		}

		// Accumulate content
		if chunk.Content != "" {
			responseContent += chunk.Content
			streamHandler(chunk.Content)
		}

		// Collect tool calls if present
		if len(chunk.ToolCalls) > 0 {
			toolCalls = chunk.ToolCalls
		}
	}

	// If we don't have a full message yet, construct one
	if fullMessage.Role == "" {
		fullMessage = llm.Message{
			Role:      llm.RoleAssistant,
			Content:   responseContent,
			ToolCalls: toolCalls,
		}
	}

	// Add the full message to memory
	if err := a.memory.Add(fullMessage); err != nil {
		return fmt.Errorf("failed to add assistant response: %w", err)
	}

	// Process tool calls if any
	if len(fullMessage.ToolCalls) > 0 && a.tools != nil {
		streamHandler("\n[Processing tool calls...]\n")

		finalResponse, err := a.handleToolCalls(ctx, fullMessage.ToolCalls)
		if err != nil {
			return err
		}

		streamHandler("\n[Final response after tool calls]\n" + finalResponse)
	}

	return nil
}

// RunConversation runs a complete conversation with multiple turns
func (a *Agent) RunConversation(ctx context.Context, userInputs []string) ([]string, error) {
	var responses []string

	for _, input := range userInputs {
		response, err := a.Run(ctx, input)
		if err != nil {
			return responses, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// EvaluateWithTools runs the agent with tools and returns detailed execution info
func (a *Agent) EvaluateWithTools(ctx context.Context, userInput string) (*AgentEvaluation, error) {
	eval := &AgentEvaluation{
		UserInput: userInput,
		Steps:     []AgentStep{},
	}

	// Add user message to memory
	if err := a.memory.Add(llm.NewUserMessage(userInput)); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// Start evaluation process
	evalStep := AgentStep{
		StepType: "initial",
	}

	// Get messages from memory
	messages, err := a.memory.Messages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}
	evalStep.InputMessages = messages

	// Check if tools are available and add them as options if so
	options := a.options
	if a.tools != nil {
		toolList := a.getToolsList()
		if len(toolList) > 0 {
			options = append(options, llm.WithTools(toolList))
		}
	}

	// Get response from LLM
	response, err := a.client.Chat(ctx, messages, options...)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	evalStep.OutputMessage = response.Message
	evalStep.TokenUsage = response.Usage
	eval.Steps = append(eval.Steps, evalStep)

	// Add the response to memory
	if err := a.memory.Add(response.Message); err != nil {
		return nil, fmt.Errorf("failed to add assistant response: %w", err)
	}

	// Check if the response contains tool calls
	if len(response.Message.ToolCalls) > 0 && a.tools != nil {
		result, steps, err := a.evaluateToolCalls(ctx, response.Message.ToolCalls)
		if err != nil {
			return nil, err
		}

		eval.Steps = append(eval.Steps, steps...)
		eval.FinalResponse = result
	} else {
		eval.FinalResponse = response.Message.Content
	}

	return eval, nil
}

// evaluateToolCalls processes tool calls and records evaluation steps
func (a *Agent) evaluateToolCalls(ctx context.Context, toolCalls []llm.ToolCall) (string, []AgentStep, error) {
	var steps []AgentStep

	// Process each tool call
	toolStep := AgentStep{
		StepType:  "tool_execution",
		ToolCalls: toolCalls,
	}

	var toolResponses []llm.Message
	for _, tc := range toolCalls {
		// Call the tool
		toolResponse, err := a.tools.Call(ctx, tc)
		if err != nil {
			return "", steps, fmt.Errorf("tool execution error: %w", err)
		}

		toolResponses = append(toolResponses, toolResponse)

		// Add tool response to memory
		if err := a.memory.Add(toolResponse); err != nil {
			return "", steps, fmt.Errorf("failed to add tool response: %w", err)
		}
	}

	toolStep.ToolResponses = toolResponses
	steps = append(steps, toolStep)

	// Get messages from memory
	messages, err := a.memory.Messages()
	if err != nil {
		return "", steps, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Get next response from LLM with tool results
	responseStep := AgentStep{
		StepType:      "response",
		InputMessages: messages,
	}

	options := a.options
	options = append(options, llm.WithToolChoice(nil))
	if a.tools != nil {
		toolList := a.getToolsList()
		if len(toolList) > 0 {
			options = append(options, llm.WithTools(toolList))
			options = append(options, llm.WithToolChoice("auto"))
		}
	}

	response, err := a.client.Chat(ctx, messages, options...)
	if err != nil {
		return "", steps, fmt.Errorf("LLM error: %w", err)
	}

	responseStep.OutputMessage = response.Message
	responseStep.TokenUsage = response.Usage
	steps = append(steps, responseStep)

	// Add the response to memory
	if err := a.memory.Add(response.Message); err != nil {
		return "", steps, fmt.Errorf("failed to add assistant response: %w", err)
	}

	// Check if we have more tool calls to handle
	if len(response.Message.ToolCalls) > 0 {
		result, moreSteps, err := a.evaluateToolCalls(ctx, response.Message.ToolCalls)
		if err != nil {
			return "", steps, err
		}

		steps = append(steps, moreSteps...)
		return result, steps, nil
	}

	return response.Message.Content, steps, nil
}

// Types for evaluation

type AgentEvaluation struct {
	UserInput     string      `json:"user_input"`
	Steps         []AgentStep `json:"steps"`
	FinalResponse string      `json:"final_response"`
}

type AgentStep struct {
	StepType      string         `json:"step_type"`      // "initial", "tool_execution", "response"
	InputMessages []llm.Message  `json:"input_message"`  // Messages sent to the LLM
	OutputMessage llm.Message    `json:"output_message"` // Response from the LLM
	ToolCalls     []llm.ToolCall `json:"tool_calls"`     // Tool calls made
	ToolResponses []llm.Message  `json:"tool_responses"` // Responses from the tools
	TokenUsage    llm.Usage      `json:"token_usage"`    // Token usage information
}
