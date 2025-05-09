package llm

import (
	"context"
)

// LLM represents a generic large language model interface
type LLM interface {
	// Chat generates a response based on the conversation history
	Chat(ctx context.Context, messages []Message, opts ...Option) (Response, error)

	// ChatStream streams the response tokens
	ChatStream(ctx context.Context, messages []Message, opts ...Option) (Stream, error)
}

// Response contains the model's response and additional metadata
type Response struct {
	Message Message
	Usage   Usage
}

// Stream represents a streaming response
type Stream interface {
	// Next returns the next chunk of the stream
	// Returns io.EOF when the stream is complete
	Next() (Message, error)

	// Close closes the stream
	Close() error
}

// Client represents a configured LLM client
type Client struct {
	llm LLM
}

// NewClient creates a new LLM client
func NewClient(llm LLM) *Client {
	return &Client{llm: llm}
}

// Chat generates a response based on the conversation history
func (c *Client) Chat(ctx context.Context, messages []Message, opts ...Option) (Response, error) {
	return c.llm.Chat(ctx, messages, opts...)
}

// ChatStream streams the response tokens
func (c *Client) ChatStream(ctx context.Context, messages []Message, opts ...Option) (Stream, error) {
	return c.llm.ChatStream(ctx, messages, opts...)
}
