package ocr

import (
	"context"
)

// OCRProvider represents an interface for OCR operations
type OCRProvider interface {
	// ExtractText extracts text from an image
	ExtractText(ctx context.Context, imageData []byte, opts ...Option) (Result, error)

	// ExtractTextFromURL extracts text from an image at the given URL
	ExtractTextFromURL(ctx context.Context, imageURL string, opts ...Option) (Result, error)
}

// Result represents the output of an OCR operation
type Result struct {
	// Text is the extracted text
	Text string

	// Confidence is the overall confidence score (0-1)
	Confidence float32

	// Blocks contains detailed information about text blocks (if supported by provider)
	Blocks []TextBlock

	// Usage contains token/resource usage statistics
	Usage Usage
}

// TextBlock represents a block of text detected in the image
type TextBlock struct {
	// Text is the content of this block
	Text string

	// Confidence is the confidence score for this block (0-1)
	Confidence float32

	// BoundingBox represents the location of the text in the image (if available)
	BoundingBox BoundingBox
}

// BoundingBox represents the position of text in an image
type BoundingBox struct {
	X      float32 // Left coordinate (normalized 0-1)
	Y      float32 // Top coordinate (normalized 0-1)
	Width  float32 // Width (normalized 0-1)
	Height float32 // Height (normalized 0-1)
}

// Usage represents resource usage statistics for OCR operations
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	ProcessingTime   int // in milliseconds
}

// Client represents a configured OCR client
type Client struct {
	provider OCRProvider
}

// NewClient creates a new OCR client
func NewClient(provider OCRProvider) *Client {
	return &Client{provider: provider}
}

// ExtractText extracts text from an image
func (c *Client) ExtractText(ctx context.Context, imageData []byte, opts ...Option) (Result, error) {
	return c.provider.ExtractText(ctx, imageData, opts...)
}

// ExtractTextFromURL extracts text from an image at the given URL
func (c *Client) ExtractTextFromURL(ctx context.Context, imageURL string, opts ...Option) (Result, error) {
	return c.provider.ExtractTextFromURL(ctx, imageURL, opts...)
}

