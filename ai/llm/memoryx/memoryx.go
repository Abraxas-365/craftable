package memoryx

import (
	"github.com/Abraxas-365/craftable/ai/llm"
)

// Memory represents a conversation memory with system prompt management
type Memory interface {
	// Messages returns all messages including system prompt
	// May return error if retrieval fails (e.g., database error)
	Messages() ([]llm.Message, error)

	// Add adds a new message to memory
	// Returns error if the operation fails
	Add(message llm.Message) error

	// Clear resets the conversation but keeps the system prompt
	// Returns error if the operation fails
	Clear() error

	// SystemPrompt gets the current system prompt
	// May return error if retrieval fails
	SystemPrompt() (string, error)

	// UpdateSystemPrompt updates the system prompt
	// Returns error if the operation fails
	UpdateSystemPrompt(content string) error
}

// MemoryOption configures the memory
type MemoryOption func(*DefaultMemory)

// WithMaxMessages sets the maximum number of messages to retain
func WithMaxMessages(max int) MemoryOption {
	return func(m *DefaultMemory) {
		m.maxMessages = max
	}
}

// DefaultMemory implements the Memory interface with in-memory storage
type DefaultMemory struct {
	systemPrompt string
	messages     []llm.Message
	maxMessages  int
}

// NewMemory creates a new memory instance with a system prompt
func NewMemory(systemPrompt string, opts ...MemoryOption) *DefaultMemory {
	m := &DefaultMemory{
		systemPrompt: systemPrompt,
		messages:     []llm.Message{llm.NewSystemMessage(systemPrompt)},
		maxMessages:  100,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// In-memory implementation of the Memory interface

func (m *DefaultMemory) Messages() ([]llm.Message, error) {
	return m.messages, nil
}

func (m *DefaultMemory) Add(message llm.Message) error {
	m.messages = append(m.messages, message)

	if len(m.messages) > m.maxMessages {
		excess := len(m.messages) - m.maxMessages
		m.messages = append([]llm.Message{m.messages[0]}, m.messages[excess+1:]...)
	}

	return nil
}

func (m *DefaultMemory) Clear() error {
	m.messages = []llm.Message{llm.NewSystemMessage(m.systemPrompt)}
	return nil
}

func (m *DefaultMemory) SystemPrompt() (string, error) {
	return m.systemPrompt, nil
}

func (m *DefaultMemory) UpdateSystemPrompt(content string) error {
	m.systemPrompt = content
	if len(m.messages) > 0 && m.messages[0].Role == llm.RoleSystem {
		m.messages[0] = llm.NewSystemMessage(content)
	} else {
		m.messages = append([]llm.Message{llm.NewSystemMessage(content)}, m.messages...)
	}
	return nil
}
