package embedding

import (
	"context"
)

// Embedder represents an interface for text embedding operations
type Embedder interface {
	// EmbedDocuments converts a slice of documents into vector embeddings
	EmbedDocuments(ctx context.Context, documents []string, opts ...Option) ([]Embedding, error)

	// EmbedQuery converts a single query text into a vector embedding
	EmbedQuery(ctx context.Context, text string, opts ...Option) (Embedding, error)
}

// Embedding represents a vector embedding result
type Embedding struct {
	// Vector is the embedding vector
	Vector []float32

	// Usage contains token usage statistics
	Usage Usage
}

// Usage represents token usage statistics for embeddings
type Usage struct {
	PromptTokens int
	TotalTokens  int
}

// Client represents a configured embedding client
type Client struct {
	embedder Embedder
}

// NewClient creates a new embedding client
func NewClient(embedder Embedder) *Client {
	return &Client{embedder: embedder}
}

// EmbedDocuments converts a slice of documents into vector embeddings
func (c *Client) EmbedDocuments(ctx context.Context, documents []string, opts ...Option) ([]Embedding, error) {
	return c.embedder.EmbedDocuments(ctx, documents, opts...)
}

// EmbedQuery converts a single query text into a vector embedding
func (c *Client) EmbedQuery(ctx context.Context, text string, opts ...Option) (Embedding, error) {
	return c.embedder.EmbedQuery(ctx, text, opts...)
}
