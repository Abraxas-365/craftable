package llm

// Role constants
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleFunction  = "function"
	RoleTool      = "tool"
)

// Message represents a chat message
type Message struct {
	Role         string         `json:"role"`
	Content      string         `json:"content,omitempty"`
	Name         string         `json:"name,omitempty"`
	FunctionCall *FunctionCall  `json:"function_call,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// FunctionCall represents a function call in a message
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Function describes a callable function
type Function struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema object
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// Tool represents a callable tool
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

// NewSystemMessage creates a new system message
func NewSystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

// NewFunctionMessage creates a new function message
func NewFunctionMessage(name string, content string) Message {
	return Message{
		Role:    RoleFunction,
		Name:    name,
		Content: content,
	}
}

// NewToolMessage creates a new tool message
func NewToolMessage(toolCallID string, content string) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Content:    content,
	}
}

// ResponseFormatType represents the format type for model outputs
type ResponseFormatType string

const (
	// JSONObject requests output in JSON object format
	JSONObject ResponseFormatType = "json_object"
	// TextFormat requests output in plain text (default)
	TextFormat ResponseFormatType = "text"
	// JSONSchema requests output conforming to a specific JSON schema
	JSONSchema ResponseFormatType = "json_schema"
)

// ResponseFormat specifies the desired output format
type ResponseFormat struct {
	Type       ResponseFormatType `json:"type"`
	JSONSchema any                `json:"schema,omitempty"` // Optional JSON schema for JSONSchema type
}

// WithResponseFormat specifies the output format
func WithResponseFormat(format *ResponseFormat) Option {
	return func(o *ChatOptions) {
		o.ResponseFormat = format
	}
}

// WithJSONResponseFormat sets the response format to JSON object
func WithJSONResponseFormat() Option {
	return func(o *ChatOptions) {
		o.ResponseFormat = &ResponseFormat{
			Type: JSONObject,
		}
	}
}

// WithJSONSchemaResponseFormat sets the response format to conform to a specific JSON schema
func WithJSONSchemaResponseFormat(schema any) Option {
	return func(o *ChatOptions) {
		o.ResponseFormat = &ResponseFormat{
			Type:       JSONSchema,
			JSONSchema: schema,
		}
	}
}
